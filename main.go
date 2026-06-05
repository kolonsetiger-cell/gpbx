package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"gitee.com/kolonse_zhjsh/gpbx/app"

	"gitee.com/kolonse_zhjsh/gpbx/log"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/auth"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/notify/esl_notify"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/notify/http_notify"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/ai"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/callcenter_agent"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/callcenter_api"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/callcenter_manager"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/ivr"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/monitor"
	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/router"

	_ "gitee.com/kolonse_zhjsh/gpbx/modules/server/telegram_bot"
	"github.com/sevlyar/go-daemon"
)

var enable_daemon bool
var config_path string = "./config.kcfg"
var exit_sig chan os.Signal = make(chan os.Signal, 1)

func serve() {
	app.GetDefaultApp().SetLogger(log.NormalLogger)
	err := app.GetDefaultApp().Init(config_path)
	if err != nil {
		panic(err)
	}
	err = app.GetDefaultApp().Run()
	if err != nil {
		panic(err)
	}
	err = app.GetDefaultApp().Uninit()
	if err != nil {
		panic(err)
	}
}

func get_curdir() (string, error) {
	exePath := os.Args[0]
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return "", err
	}
	// 获取所在目录
	dir := filepath.Dir(absPath)
	return dir, nil
}

// deriveAESKey 从 SHA512 hex 字符串推导 AES-256 key（取前32字节）
func deriveAESKey(sha512hex string) ([]byte, error) {
	hashBytes, err := hex.DecodeString(sha512hex)
	if err != nil {
		return nil, fmt.Errorf("解码 sign_key hex 失败: %w", err)
	}
	if len(hashBytes) != 64 {
		return nil, fmt.Errorf("SHA512 hash 长度异常: %d", len(hashBytes))
	}
	return hashBytes[:32], nil
}

// aesDecrypt AES-256-GCM 解密
func aesDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文太短")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aesGCM.Open(nil, nonce, ct, nil)
}

const (
	separator = "#####"
)

type ifaceInfo struct {
	IP  string
	MAC string
}

// getMachineHash 获取本机所有网卡IPv4+MAC，按IP排序，生成 SHA512 hex
func getMachineHash() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var infos []ifaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			// fmt.Printf("[跳过] 网卡 %s (未启用)\n", iface.Name)
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			// fmt.Printf("[跳过] 网卡 %s (获取地址失败: %v)\n", iface.Name, err)
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				// fmt.Printf("[跳过] 网卡 %s 地址 %s (非IPv4)\n", iface.Name, ipNet.IP.String())
				continue
			}
			mac := iface.HardwareAddr.String()
			if mac == "" {
				mac = "00:00:00:00:00:00"
			}
			// fmt.Printf("[采集] 网卡 %s  IP=%s  MAC=%s\n", iface.Name, ip.String(), mac)
			infos = append(infos, ifaceInfo{IP: ip.String(), MAC: mac})
		}
	}

	if len(infos) == 0 {
		return "", fmt.Errorf("未找到有效网卡")
	}

	// 按 IP 排序
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].IP < infos[j].IP
	})

	// fmt.Println("=== 按IP排序后的网卡列表 ===")
	var parts []string
	for _, info := range infos {
		fmt.Printf("  %s:%s\n", info.IP, info.MAC)
		parts = append(parts, fmt.Sprintf("%s:%s", info.IP, info.MAC))
	}
	str := strings.Join(parts, ";")
	// fmt.Printf("拼接字符串: %s\n", str)
	hash := sha512.Sum512([]byte(str))
	hexHash := hex.EncodeToString(hash[:])
	// fmt.Printf("SHA512: %s\n", hexHash)
	return hexHash, nil
}

// rsaDecryptPublic 使用公钥解密: m = c^e mod n
func rsaDecryptPublic(data []byte, pub *rsa.PublicKey) ([]byte, error) {
	c := new(big.Int).SetBytes(data)
	e := big.NewInt(int64(pub.E))
	m := new(big.Int).Exp(c, e, pub.N)
	return m.Bytes(), nil
}

// aesEncrypt AES-256-GCM 加密
func aesEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

var cstZone = time.FixedZone("CST", 8*3600) // UTC+8

// nowCST 获取当前时间，如果系统时区是 UTC，则强制转为 CST
func nowCST() time.Time {
	now := time.Now()
	_, offset := now.Zone()
	if offset == 0 {
		return now.In(cstZone)
	}
	return now
}

func cmdInfo() error {
	hash, err := getMachineHash()
	if err != nil {
		return fmt.Errorf("获取机器信息失败: %w", err)
	}
	if err := os.WriteFile("kolonse.out", []byte(hash), 0644); err != nil {
		return fmt.Errorf("写入 kolonse.out 失败: %w", err)
	}
	fmt.Println("kolonse.out 已生成（机器签名）")
	return nil
}

func cmdReg() error {
	// 读取 kolonse.sign
	encrypted, err := os.ReadFile("kolonse.sign")
	if err != nil {
		return fmt.Errorf("读取 kolonse.sign 失败: %w", err)
	}

	// 读取公钥
	pubPEM, err := os.ReadFile("kolonse.pub")
	if err != nil {
		return fmt.Errorf("读取 kolonse.pub 失败: %w", err)
	}
	block, _ := pem.Decode(pubPEM)
	if block == nil {
		return fmt.Errorf("解析公钥 PEM 失败")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("解析公钥失败: %w", err)
	}
	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("公钥类型错误")
	}

	// 公钥解密
	decrypted, err := rsaDecryptPublic(encrypted, pubKey)
	if err != nil {
		return fmt.Errorf("公钥解密失败: %w", err)
	}
	// 拆分为 content[3]: hash, time, key
	content := strings.Split(string(decrypted), separator)
	if len(content) < 3 {
		return fmt.Errorf("解密内容格式错误: 期望3部分，实际%d部分", len(content))
	}

	// 验证签名: 重新获取机器签名与 content[0] 对比
	currentHash, err := getMachineHash()
	if err != nil {
		return fmt.Errorf("获取当前机器签名失败: %w", err)
	}
	if currentHash != content[0] {
		fmt.Println("签名不一致", currentHash, content[0])
		os.Exit(1)
	}

	// 验证时间: 与 content[1] 相差不超过10分钟（统一使用 CST 时区）
	signTime, err := time.ParseInLocation("2006-01-02 15:04:05", content[1], cstZone)
	if err != nil {
		return fmt.Errorf("解析签名时间失败: %w", err)
	}
	now := nowCST()
	diff := now.Sub(signTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > 10*time.Minute {
		fmt.Println("签名过期")
		os.Exit(1)
	}

	// 构建 sign_string: <kolonse.sign hex>#####<当前时间>
	nowStr := now.Format("2006-01-02 15:04:05")
	signHex := hex.EncodeToString(encrypted)
	signString := fmt.Sprintf("%s%s%s", signHex, separator, nowStr)

	// 构建 sign_key: SHA512(<content[2]>#####<当前时间>)  hex
	signKeyRaw := fmt.Sprintf("%s%s%s", content[2], separator, nowStr)
	signKeyHash := sha512.Sum512([]byte(signKeyRaw))
	signKey := hex.EncodeToString(signKeyHash[:])

	// 保存 kolonse.key
	if err := os.WriteFile("kolonse.key", []byte(signKey), 0644); err != nil {
		return fmt.Errorf("写入 kolonse.key 失败: %w", err)
	}

	// AES-256 加密 sign_string
	aesKey, err := deriveAESKey(signKey)
	if err != nil {
		return fmt.Errorf("推导AES密钥失败: %w", err)
	}
	certData, err := aesEncrypt(aesKey, []byte(signString))
	if err != nil {
		return fmt.Errorf("AES加密失败: %w", err)
	}
	if err := os.WriteFile("kolonse.cert", certData, 0644); err != nil {
		return fmt.Errorf("写入 kolonse.cert 失败: %w", err)
	}

	fmt.Println("注册成功: kolonse.key / kolonse.cert 已生成")
	return nil
}
func cmdDefault() error {
	// 读取 kolonse.cert
	certData, err := os.ReadFile("kolonse.cert")
	if err != nil {
		return fmt.Errorf("读取 kolonse.cert 失败: %w", err)
	}

	// 读取 kolonse.key (sign_key hex)
	signKeyBytes, err := os.ReadFile("kolonse.key")
	if err != nil {
		return fmt.Errorf("读取 kolonse.key 失败: %w", err)
	}
	signKey := strings.TrimSpace(string(signKeyBytes))

	// 推导 AES key
	aesKey, err := deriveAESKey(signKey)
	if err != nil {
		return fmt.Errorf("推导AES密钥失败: %w", err)
	}

	// AES 解密
	decrypted, err := aesDecrypt(aesKey, certData)
	if err != nil {
		return fmt.Errorf("AES解密失败: %w", err)
	}

	// 拆分 sign_string: <signHex>#####<timestamp>
	content := strings.Split(string(decrypted), separator)
	if len(content) < 2 {
		return fmt.Errorf("证书内容格式错误")
	}
	sign_hex := content[0]
	sign_raw, _ := hex.DecodeString(sign_hex)
	pubPEM, err := os.ReadFile("kolonse.pub")
	if err != nil {
		return fmt.Errorf("读取 kolonse.pub 失败: %w", err)
	}
	block, _ := pem.Decode(pubPEM)
	if block == nil {
		return fmt.Errorf("解析公钥 PEM 失败")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("解析公钥失败: %w", err)
	}
	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("公钥类型错误")
	}

	// 公钥解密
	sign_src, err := rsaDecryptPublic(sign_raw, pubKey)
	if err != nil {
		return fmt.Errorf("公钥解密失败: %w", err)
	}
	content2 := strings.Split(string(sign_src), separator)
	if len(content2) < 2 {
		return fmt.Errorf("证书内容格式错误")
	}
	sign_hash := content2[0]
	// 重新获取机器签名，与 content[0] (即 signHex) 对比
	// content[0] 是 kolonse.sign 的 hex 编码，不是原始哈希，需要额外验证
	currentHash, err := getMachineHash()
	if err != nil {
		return fmt.Errorf("获取当前机器签名失败: %w", err)
	}

	if currentHash != sign_hash {
		return fmt.Errorf("签名不一致")
	}
	return nil
}

func main() {
	flagInfo := flag.Bool("info", false, "获取机器签名")
	flagReg := flag.Bool("reg", false, "注册验证并生成证书")
	flag.BoolVar(&enable_daemon, "D", false, "-D daemon,not support windows")
	flag.StringVar(&config_path, "conf", "./config.kcfg", "-conf config.kcfg")
	flag.Parse()

	if *flagInfo {
		_ = cmdInfo()
		return
	}
	if *flagReg {
		err := cmdReg()
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	err := cmdDefault()
	if err != nil {
		panic(err)
	}
	if runtime.GOOS == "windows" {
		enable_daemon = false
	}

	if !enable_daemon {
		signal.Notify(exit_sig, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			_, ok := <-exit_sig
			if ok {
				app.GetDefaultApp().TriggerExit()
			}
		}()
		serve()
		signal.Stop(exit_sig)
		return
	}
	dir, err := get_curdir()
	if err != nil {
		panic(err)
	}
	err = os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	cntxt := &daemon.Context{
		PidFileName: "gpbx.pid",
		PidFilePerm: 0644,
		WorkDir:     dir,
		Umask:       027,
		Args:        []string{"gpbx daemon"},
	}

	d, err := cntxt.Reborn()
	if err != nil {
		panic(err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()
	daemon.SetSigHandler(func(sig os.Signal) error {
		app.GetDefaultApp().TriggerExit()
		return nil
	}, syscall.SIGINT, syscall.SIGTERM)
	go serve()
	_ = daemon.ServeSignals()
}
