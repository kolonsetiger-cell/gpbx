import http.server
import json
import re
from http import HTTPStatus

# 模拟外援部人员数据（100人简化模拟，覆盖四种风格类型、不同年龄范围）
SIMULATE_PERSONNEL = [
    # 萝莉类型（loli）：18~22岁
    {"name": "张三", "age": 19, "weight": "45kg", "phone": "138XXXX1234"},
    {"name": "李四", "age": 20, "weight": "48kg", "phone": "139XXXX5678"},
    {"name": "赵六", "age": 21, "weight": "46kg", "phone": "136XXXX7890"},
    # 少妇类型（married_woman）：28~35岁
    {"name": "王芳", "age": 30, "weight": "55kg", "phone": "135XXXX2345"},
    {"name": "李娜", "age": 32, "weight": "53kg", "phone": "134XXXX6789"},
    # 御姐类型（royal_sister）：25~30岁
    {"name": "张敏", "age": 26, "weight": "52kg", "phone": "137XXXX3456"},
    {"name": "刘婷", "age": 28, "weight": "54kg", "phone": "132XXXX8901"},
    # 清纯少女类型（pure_girl）：18~25岁
    {"name": "陈静", "age": 22, "weight": "49kg", "phone": "131XXXX4567"},
    {"name": "杨雪", "age": 23, "weight": "50kg", "phone": "130XXXX9012"},
    # 其余模拟数据（简化，实际可扩展至100人，按上述格式补充）
]

# 支持的风格类型
SUPPORTED_STYLES = {"loli", "married_woman", "royal_sister", "pure_girl"}

class PairingHandler(http.server.BaseHTTPRequestHandler):
    # 解析JSON请求体
    def _parse_request_body(self):
        try:
            content_length = int(self.headers.get("Content-Length", 0))
            if content_length == 0:
                return None
            body = self.rfile.read(content_length)
            return json.loads(body)
        except json.JSONDecodeError:
            return None

    # 生成响应数据
    def _send_response(self, status_code, data):
        self.send_response(status_code)
        self.send_header("Content-type", "application/json; charset=utf-8")
        self.end_headers()
        self.wfile.write(json.dumps(data, ensure_ascii=False).encode("utf-8"))

    # 处理POST请求（Dify工具调用仅用POST）
    def do_POST(self):
        # 仅处理指定接口路径，其余路径返回404
        if self.path != "/api/query":
            self._send_response(HTTPStatus.NOT_FOUND, {
                "code": 404,
                "message": "接口不存在，请访问/api/query",
                "data": []
            })
            return

        # 解析请求参数
        request_data = self._parse_request_body()
        if not request_data:
            self._send_response(HTTPStatus.BAD_REQUEST, {
                "code": 400,
                "message": "请求参数格式错误，需为JSON格式",
                "data": []
            })
            return

        # 校验必填参数
        required_params = ["age_range", "style_type"]
        if not all(param in request_data for param in required_params):
            self._send_response(HTTPStatus.BAD_REQUEST, {
                "code": 400,
                "message": "缺少必填参数，需传入age_range（年龄范围）和style_type（风格类型）",
                "data": []
            })
            return

        age_range = request_data["age_range"]
        style_type = request_data["style_type"]

        # 校验风格类型
        if style_type not in SUPPORTED_STYLES:
            self._send_response(HTTPStatus.BAD_REQUEST, {
                "code": 400,
                "message": f"风格类型不支持，仅支持{list(SUPPORTED_STYLES)}",
                "data": []
            })
            return

        # 校验年龄范围格式（必须为X~Y格式，X、Y为正整数）
        age_pattern = re.compile(r"^\d+~\d+$")
        if not age_pattern.match(age_range):
            self._send_response(HTTPStatus.BAD_REQUEST, {
                "code": 400,
                "message": "年龄范围格式错误，需为\"X~Y\"（如18~20）",
                "data": []
            })
            return

        # 解析年龄范围
        min_age, max_age = map(int, age_range.split("~"))
        if min_age >= max_age:
            self._send_response(HTTPStatus.BAD_REQUEST, {
                "code": 400,
                "message": "年龄范围错误，最小值需小于最大值",
                "data": []
            })
            return

        # 模拟查询：筛选符合条件的人员
        matched_personnel = []
        for person in SIMULATE_PERSONNEL:
            if person["age"] >= min_age and person["age"] <= max_age:
                # 根据风格类型匹配（此处简化，实际可给每个模拟人员添加style_type字段精准匹配）
                # 按风格类型对应年龄区间模拟匹配，贴合实际场景
                if style_type == "loli" and 18 <= person["age"] <= 22:
                    matched_personnel.append(person)
                elif style_type == "married_woman" and 28 <= person["age"] <= 35:
                    matched_personnel.append(person)
                elif style_type == "royal_sister" and 25 <= person["age"] <= 30:
                    matched_personnel.append(person)
                elif style_type == "pure_girl" and 18 <= person["age"] <= 25:
                    matched_personnel.append(person)

        # 返回响应结果（完全适配之前定义的格式）
        if matched_personnel:
            self._send_response(HTTPStatus.OK, {
                "code": 200,
                "message": "查询成功",
                "data": matched_personnel
            })
        else:
            self._send_response(HTTPStatus.NOT_FOUND, {
                "code": 404,
                "message": "未找到符合条件的人员",
                "data": []
            })

def run_server(port=8083):
    server_address = ("0.0.0.0", port)
    httpd = http.server.HTTPServer(server_address, PairingHandler)
    print(f"Python HTTP Server 启动成功，服务地址：http://0.0.0.0:{port}")
    print(f"接口路径：http://0.0.0.0:{port}/api/query")
    print("按Ctrl+C停止服务...")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\n服务正在停止...")
        httpd.server_close()
        print("服务已停止")

if __name__ == "__main__":
    # 可修改端口，默认8080（与Dify工具配置的服务器地址端口一致）
    run_server(port=8083)
    