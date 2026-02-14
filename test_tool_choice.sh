#!/bin/bash

# 测试 tool_choice 参数是否生效

echo "=== 测试1: 直接调用 LLM API 验证 tool_choice 支持 ==="
echo ""

curl -s https://llm.ooxo.cc/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5-1.5b-instruct",
    "messages": [{"role": "user", "content": "北京今天天气怎么样"}],
    "tools": [{
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "获取天气信息",
            "parameters": {
                "type": "object",
                "properties": {
                    "location": {"type": "string", "description": "城市名称"}
                },
                "required": ["location"]
            }
        }
    }],
    "tool_choice": "required"
}' | jq '.choices[0].message.tool_calls'

echo ""
echo "=== 测试2: 不使用 tool_choice (auto) ==="
echo ""

curl -s https://llm.ooxo.cc/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5-1.5b-instruct",
    "messages": [{"role": "user", "content": "北京今天天气怎么样"}],
    "tools": [{
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "获取天气信息",
            "parameters": {
                "type": "object",
                "properties": {
                    "location": {"type": "string"}
                },
                "required": ["location"]
            }
        }
    }],
    "tool_choice": "auto"
}' | jq '.choices[0].message | {content, tool_calls}'

echo ""
echo "测试完成！"
echo ""
echo "说明："
echo "- 测试1 使用 tool_choice: required，应该强制调用工具"
echo "- 测试2 使用 tool_choice: auto，LLM 可以选择是否调用工具"
