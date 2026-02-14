#!/bin/bash

# 测试新的RAG Tool Calling功能

echo "测试RAG问答（AI自主决定搜索策略）..."
echo ""

curl -N -X POST http://localhost:8000/v1/articles/rag/question \
  -H "Content-Type: application/json" \
  -d '{
    "question": "2026年的Node版本管理器有哪些选择"
  }'

echo ""
echo "测试完成"
