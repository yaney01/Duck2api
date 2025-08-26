#!/usr/bin/env python3
import requests
import json
import time

def test_duck2api():
    """测试Duck2api服务是否正常工作"""
    base_url = "http://localhost:8080"
    
    # 测试模型列表端点
    print("测试模型列表端点...")
    try:
        response = requests.get(f"{base_url}/v1/models", timeout=10)
        if response.status_code == 200:
            models = response.json()
            print(f"✓ 模型列表获取成功，共有 {len(models.get('data', []))} 个模型")
            for model in models.get('data', []):
                print(f"  - {model.get('id')}")
        else:
            print(f"✗ 模型列表获取失败，状态码: {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"✗ 请求模型列表时出错: {e}")
    
    print("\n" + "="*50 + "\n")
    
    # 测试聊天完成端点（非流式）
    print("测试聊天完成端点（非流式）...")
    chat_data = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "Hello, please say 'test successful' if you can respond."}
        ],
        "stream": False
    }
    
    try:
        response = requests.post(
            f"{base_url}/v1/chat/completions",
            json=chat_data,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        if response.status_code == 200:
            result = response.json()
            if 'choices' in result and len(result['choices']) > 0:
                message = result['choices'][0]['message']['content']
                print(f"✓ 聊天接口工作正常")
                print(f"响应: {message[:100]}...")
            else:
                print("✗ 响应格式异常")
                print(f"响应内容: {result}")
        else:
            print(f"✗ 聊天接口请求失败，状态码: {response.status_code}")
            print(f"错误信息: {response.text}")
    except requests.exceptions.RequestException as e:
        print(f"✗ 请求聊天接口时出错: {e}")
    
    print("\n" + "="*50 + "\n")
    
    # 测试流式响应
    print("测试聊天完成端点（流式）...")
    chat_data_stream = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "Count from 1 to 5"}
        ],
        "stream": True
    }
    
    try:
        response = requests.post(
            f"{base_url}/v1/chat/completions",
            json=chat_data_stream,
            headers={"Content-Type": "application/json"},
            stream=True,
            timeout=30
        )
        
        if response.status_code == 200:
            print("✓ 流式响应开始...")
            chunks_received = 0
            for line in response.iter_lines():
                if line:
                    line_str = line.decode('utf-8')
                    if line_str.startswith('data: '):
                        chunks_received += 1
                        if chunks_received <= 3:  # 只显示前3个块
                            print(f"  接收到数据块: {line_str[:80]}...")
                    if '[DONE]' in line_str:
                        break
            print(f"✓ 流式响应完成，共接收 {chunks_received} 个数据块")
        else:
            print(f"✗ 流式请求失败，状态码: {response.status_code}")
            print(f"错误信息: {response.text}")
    except requests.exceptions.RequestException as e:
        print(f"✗ 请求流式接口时出错: {e}")

if __name__ == "__main__":
    print("Duck2api 功能测试")
    print("请确保Duck2api服务正在localhost:8080上运行")
    print("="*50)
    test_duck2api()