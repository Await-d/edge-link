#!/usr/bin/env python3
"""
简单的Webhook接收器用于测试Edge-Link告警通知
"""
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import sys

class WebhookHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)

        print("\n" + "="*80)
        print("✅ 收到Webhook通知！")
        print("="*80)

        try:
            data = json.loads(post_data.decode('utf-8'))
            print(f"\n告警ID: {data.get('alert_id')}")
            print(f"标题: {data.get('title')}")
            print(f"消息: {data.get('message')}")
            print(f"严重程度: {data.get('severity')}")
            print(f"告警类型: {data.get('alert_type')}")
            print(f"设备ID: {data.get('device_id', 'N/A')}")
            print(f"创建时间: {data.get('created_at')}")
            print(f"时间戳: {data.get('timestamp')}")

            if data.get('metadata'):
                print("\n元数据:")
                for key, value in data['metadata'].items():
                    print(f"  {key}: {value}")

            print("\n" + "="*80 + "\n")

        except Exception as e:
            print(f"解析错误: {e}")
            print(f"原始数据: {post_data.decode('utf-8')}")

        # 返回200 OK
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(b'{"status": "received"}')

    def log_message(self, format, *args):
        # 静默日志
        pass

def run_server(port=8888):
    server_address = ('', port)
    httpd = HTTPServer(server_address, WebhookHandler)
    print(f"🚀 Webhook接收器启动在端口 {port}")
    print(f"📡 Webhook URL: http://localhost:{port}/webhook")
    print("⏳ 等待告警通知...\n")
    httpd.serve_forever()

if __name__ == '__main__':
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8888
    try:
        run_server(port)
    except KeyboardInterrupt:
        print("\n\n👋 接收器已停止")
