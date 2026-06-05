from flask import Flask, jsonify, request
import requests
from pydantic import BaseModel

app = Flask(__name__)

task=[{
    "id": 1683,
    "taskid": "127149",
    "prologue": [
        "https://gzqkaipublicwav.gzquankeinfo.com:15180/voice/personal/0318/6007-7.wav",
        "{name=zh_ca;voice=jiajia;vol=0;speed=-1}测试医院",
        "https://gzqkaipublicwav.gzquankeinfo.com:15180/voice/personal/0318/6007-8.wav",
        "{name=zh_ca;voice=jiajia;vol=0;speed=-1}陈小美",
        "https://gzqkaipublicwav.gzquankeinfo.com:15180/voice/personal/0318/6011-1.wav"
    ],
    "asrcallback": "http://127.0.0.1:8091/api/callback/asrAccept",
    "sections": "4=08:30:00-11:30:00;4=14:00:00-17:30:00;4=18:00:00-20:00:00;",
    "displayNo": "87829910",
    "appkey": "zh_ca",
    "VocabularyId": "",
    "CustomizationId": "",
    "maxSentenceSilence": 1800,
    "phones": "10000",
    "recognition_account": "zh_caTTS"
}]

@app.route('/robot/asr/notify', methods=['POST','GET'])
def data():
    return jsonify({"code": 0, "message": "Success"}), 200

@app.route('/task/get', methods=['POST','GET'])
def data2():
    return jsonify(task), 200

@app.route('/session/create', methods=['POST','GET'])
def session_create():
    data = request.get_json()
    print("session_create", data)
    # session_id = data.get("session_id")
    # requests.post("http://127.0.0.1:50055/cti/robot/play/tts", json={
    #     "task_id": session_id,
    #     "text" : "this is a test"
    # })
    return jsonify({"code":0,"message":"success"}), 200

@app.route('/session/answer', methods=['POST','GET'])
def session_answer():
    data = request.get_json()
    print("session_answer", data)
    # session_id = data.get("session_id")
    # requests.post("http://127.0.0.1:50055/cti/robot/play/tts", json={
    #     "task_id": session_id,
    #     "text" : "this is a test"
    # })
    return jsonify({"code":0,"message":"success"}), 200
@app.route('/session/state', methods=['POST','GET'])
def session_state():
    data = request.get_json()
    print("session_state", data)
    # session_id = data.get("session_id")
    # requests.post("http://127.0.0.1:50055/cti/robot/play/tts", json={
    #     "task_id": session_id,
    #     "text" : "this is a test"
    # })
    return jsonify({"code":0,"message":"success"}), 200
@app.route('/session/hangup', methods=['POST','GET'])
def session_hangup():
    data = request.get_json()
    print("session_hangup", data)
    # session_id = data.get("session_id")
    # requests.post("http://127.0.0.1:50055/cti/robot/play/tts", json={
    #     "task_id": session_id,
    #     "text" : "this is a test"
    # })
    return jsonify({"code":0,"message":"success"}), 200
@app.route('/session/destroy', methods=['POST','GET'])
def session_destroy():
    data = request.get_json()
    print("session_destroy", data)
    # session_id = data.get("session_id")
    # requests.post("http://127.0.0.1:50055/cti/robot/play/tts", json={
    #     "task_id": session_id,
    #     "text" : "this is a test"
    # })
    return jsonify({"code":0,"message":"success"}), 200

@app.route('/api/check', methods=['POST','GET'])
def api_check():
    data = request.get_json()  
    print("api_check", data)
    if data.get("digits") == "123456":
        return jsonify({"code":0,"message":"success"}), 200
    return jsonify({"code":-1,"message":"failed"}), 200

@app.route('/api/check2', methods=['POST','GET'])
def api_check2():
    data = request.get_json()  
    print("api_check2", data)
    if data.get("digits") == "1234":
        return jsonify({"code":0,"message":"success"}), 200
    return jsonify({"code":-1,"message":"failed"}), 200

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8084, debug=True)