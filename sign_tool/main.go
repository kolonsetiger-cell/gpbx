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
	"sort"
	"strings"
	"time"
)

const (
	defaultKey = "b6064a15f6775c26111e60a8ce658a4d"
	separator  = "#####"
)

type ifaceInfo struct {
	IP  string
	MAC string
}

// ======================== 方法1: 获取机器签名 ========================

// getMachineHash 获取本机所有网卡IPv4+MAC，按IP排序，生成 SHA512 hex
func getMachineHash() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var infos []ifaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			fmt.Printf("[跳过] 网卡 %s (未启用)\n", iface.Name)
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Printf("[跳过] 网卡 %s (获取地址失败: %v)\n", iface.Name, err)
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				fmt.Printf("[跳过] 网卡 %s 地址 %s (非IPv4)\n", iface.Name, ipNet.IP.String())
				continue
			}
			mac := iface.HardwareAddr.String()
			if mac == "" {
				mac = "00:00:00:00:00:00"
			}
			fmt.Printf("[采集] 网卡 %s  IP=%s  MAC=%s\n", iface.Name, ip.String(), mac)
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

	fmt.Println("=== 按IP排序后的网卡列表 ===")
	var parts []string
	for _, info := range infos {
		fmt.Printf("  %s:%s\n", info.IP, info.MAC)
		parts = append(parts, fmt.Sprintf("%s:%s", info.IP, info.MAC))
	}
	str := strings.Join(parts, ";")
	fmt.Printf("拼接字符串: %s\n", str)
	hash := sha512.Sum512([]byte(str))
	hexHash := hex.EncodeToString(hash[:])
	fmt.Printf("SHA512: %s\n", hexHash)
	return hexHash, nil
}

// ======================== RSA 原始操作（私钥加密 / 公钥解密） ========================

// rsaEncryptPrivate 使用私钥加密: c = m^d mod n
func rsaEncryptPrivate(data []byte, priv *rsa.PrivateKey) ([]byte, error) {
	keySize := priv.Size()
	if len(data) > keySize-1 {
		return nil, fmt.Errorf("数据过长(%d bytes)，超过RSA单次加密上限(%d bytes)", len(data), keySize-1)
	}
	m := new(big.Int).SetBytes(data)
	c := new(big.Int).Exp(m, priv.D, priv.N)
	result := c.Bytes()
	// 左侧补零到 keySize
	padded := make([]byte, keySize)
	copy(padded[keySize-len(result):], result)
	return padded, nil
}

// rsaDecryptPublic 使用公钥解密: m = c^e mod n
func rsaDecryptPublic(data []byte, pub *rsa.PublicKey) ([]byte, error) {
	c := new(big.Int).SetBytes(data)
	e := big.NewInt(int64(pub.E))
	m := new(big.Int).Exp(c, e, pub.N)
	return m.Bytes(), nil
}

// ======================== AES-256-GCM ========================

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

// ======================== 时间工具 ========================

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

// ======================== 命令实现 ========================

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

func cmdSign() error {
	// 读取 kolonse.out
	signData, err := os.ReadFile("kolonse.out")
	if err != nil {
		return fmt.Errorf("读取 kolonse.out 失败: %w", err)
	}
	machineHash := strings.TrimSpace(string(signData))

	// 构建待签名内容: <machineHash>#####<time>#####<key>
	now := time.Now().Format("2006-01-02 15:04:05")
	content := fmt.Sprintf("%s%s%s%s%s", machineHash, separator, now, separator, defaultKey)

	// 生成 RSA 密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("生成RSA密钥失败: %w", err)
	}

	// 保存私钥
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	if err := os.WriteFile("kolonse.pri", privPEM, 0600); err != nil {
		return fmt.Errorf("保存私钥失败: %w", err)
	}

	// 保存公钥
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("序列化公钥失败: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	if err := os.WriteFile("kolonse.pub", pubPEM, 0644); err != nil {
		return fmt.Errorf("保存公钥失败: %w", err)
	}

	// 私钥加密内容
	encrypted, err := rsaEncryptPrivate([]byte(content), privateKey)
	if err != nil {
		return fmt.Errorf("私钥加密失败: %w", err)
	}

	if err := os.WriteFile("kolonse.sign", encrypted, 0644); err != nil {
		return fmt.Errorf("写入 kolonse.sign 失败: %w", err)
	}

	fmt.Println("密钥对已生成: kolonse.pri / kolonse.pub")
	fmt.Println("签名文件已生成: kolonse.sign")
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
	fmt.Println("解密内容：", string(decrypted))
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
	fmt.Println("解密内容：", string(sign_src))
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

	fmt.Printf("当前机器签名: \n%s\n%s\n", currentHash, sign_hash)
	if currentHash != sign_hash {
		return fmt.Errorf("签名不一致")
	}
	// 验证 signHex 非空即可（核心验证在 -reg 阶段已完成）
	fmt.Printf("证书时间戳: %s\n", content[1])
	fmt.Println("证书验证通过")
	return nil
}

func main() {
	flagInfo := flag.Bool("info", false, "获取机器签名")
	flagSign := flag.Bool("sign", false, "生成签名文件")
	flagReg := flag.Bool("reg", false, "注册验证并生成证书")
	flag.Parse()

	// 检查参数个数，确保互斥
	argsSet := 0
	if *flagInfo {
		argsSet++
	}
	if *flagSign {
		argsSet++
	}
	if *flagReg {
		argsSet++
	}

	if argsSet > 1 {
		fmt.Println("错误: -info, -sign, -reg 参数互斥，请只使用其中一个")
		os.Exit(1)
	}

	var err error
	switch {
	case *flagInfo:
		err = cmdInfo()
	case *flagSign:
		err = cmdSign()
	case *flagReg:
		err = cmdReg()
	default:
		err = cmdDefault()
	}

	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
}
