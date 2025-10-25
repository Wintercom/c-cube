#!/usr/bin/env python3
"""
历史客服问答批量导入服务

功能：
1. 读取转换后的问答数据
2. 通过 API 批量导入到知识库
3. 支持断点续传
4. 提供详细的导入日志
"""

import json
import sys
import time
import argparse
from typing import List, Dict, Any, Optional
from datetime import datetime
import requests
from pathlib import Path


class QABatchImporter:
    """问答批量导入器"""
    
    def __init__(self, api_url: str, token: str, knowledge_base_id: str, batch_size: int = 10):
        """
        初始化导入器
        
        Args:
            api_url: API 基础 URL
            token: 认证 token
            knowledge_base_id: 知识库 ID
            batch_size: 每批次导入数量
        """
        self.api_url = api_url.rstrip('/')
        self.token = token
        self.knowledge_base_id = knowledge_base_id
        self.batch_size = batch_size
        
        self.stats = {
            'total': 0,
            'success': 0,
            'failed': 0,
            'skipped': 0
        }
        
        self.failed_records = []
    
    def import_single_passage(self, data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        导入单条问答数据（使用 Passage API）
        
        Args:
            data: 转换后的问答数据
            
        Returns:
            API 响应结果或 None
        """
        url = f"{self.api_url}/api/v1/knowledge-bases/{self.knowledge_base_id}/knowledge/passage"
        
        headers = {
            "Authorization": f"Bearer {self.token}",
            "Content-Type": "application/json"
        }
        
        payload = {
            "passages": [data['passage']],
            "title": data.get('title', ''),
            "description": data.get('description', ''),
            "metadata": data.get('metadata', {})
        }
        
        try:
            response = requests.post(url, json=payload, headers=headers, timeout=30)
            
            if response.status_code == 201:
                return response.json()
            else:
                print(f"  ❌ API 错误 {response.status_code}: {response.text[:200]}")
                return None
                
        except requests.exceptions.Timeout:
            print("  ❌ 请求超时")
            return None
        except requests.exceptions.RequestException as e:
            print(f"  ❌ 请求失败: {e}")
            return None
    
    def import_batch(self, qa_list: List[Dict[str, Any]], start_index: int = 0) -> None:
        """
        批量导入问答数据
        
        Args:
            qa_list: 问答数据列表
            start_index: 起始索引（用于断点续传）
        """
        self.stats['total'] = len(qa_list)
        
        print(f"\n开始批量导入 (从第 {start_index + 1} 条开始)...")
        print(f"批次大小: {self.batch_size}")
        print(f"总数: {len(qa_list)} 条\n")
        
        for i in range(start_index, len(qa_list)):
            qa_data = qa_list[i]
            qa_id = qa_data.get('metadata', {}).get('qa_id', i + 1)
            title = qa_data.get('title', '')[:50]
            
            print(f"[{i + 1}/{len(qa_list)}] 导入 QA ID: {qa_id} - {title}...")
            
            try:
                result = self.import_single_passage(qa_data)
                
                if result:
                    self.stats['success'] += 1
                    print(f"  ✅ 成功")
                else:
                    self.stats['failed'] += 1
                    self.failed_records.append({
                        'index': i,
                        'qa_id': qa_id,
                        'title': title
                    })
                
            except Exception as e:
                self.stats['failed'] += 1
                print(f"  ❌ 异常: {e}")
                self.failed_records.append({
                    'index': i,
                    'qa_id': qa_id,
                    'title': title,
                    'error': str(e)
                })
            
            if (i + 1) % self.batch_size == 0:
                print(f"\n--- 已完成 {i + 1}/{len(qa_list)} 条，暂停 0.5 秒 ---\n")
                time.sleep(0.5)
            
            time.sleep(0.1)
    
    def print_stats(self):
        """打印统计信息"""
        print("\n" + "=" * 60)
        print("导入统计:")
        print(f"  总计: {self.stats['total']} 条")
        print(f"  成功: {self.stats['success']} 条")
        print(f"  失败: {self.stats['failed']} 条")
        print(f"  成功率: {self.stats['success'] / max(self.stats['total'], 1) * 100:.2f}%")
        print("=" * 60)
    
    def save_failed_records(self, output_file: str):
        """保存失败记录"""
        if not self.failed_records:
            print("\n✅ 所有记录导入成功！")
            return
        
        print(f"\n保存失败记录到: {output_file}")
        
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(self.failed_records, f, ensure_ascii=False, indent=2)
        
        print(f"已保存 {len(self.failed_records)} 条失败记录")
        print("\n失败记录详情:")
        for record in self.failed_records[:10]:
            print(f"  - [索引 {record['index']}] QA ID: {record['qa_id']} - {record['title']}")
        
        if len(self.failed_records) > 10:
            print(f"  ... 还有 {len(self.failed_records) - 10} 条")


def main():
    """主函数"""
    parser = argparse.ArgumentParser(description='批量导入历史问答数据到 C-Cube 知识库')
    
    parser.add_argument('input_file', help='转换后的 JSON 数据文件')
    parser.add_argument('--api-url', required=True, help='API 基础 URL (例如: http://localhost:8080)')
    parser.add_argument('--token', required=True, help='认证 token')
    parser.add_argument('--kb-id', required=True, help='知识库 ID')
    parser.add_argument('--batch-size', type=int, default=10, help='每批次导入数量 (默认: 10)')
    parser.add_argument('--start-index', type=int, default=0, help='起始索引，用于断点续传 (默认: 0)')
    parser.add_argument('--failed-log', default='failed_imports.json', help='失败记录保存文件 (默认: failed_imports.json)')
    
    args = parser.parse_args()
    
    print("=" * 60)
    print("C-Cube 知识库批量导入工具")
    print("=" * 60)
    print(f"输入文件: {args.input_file}")
    print(f"API URL: {args.api_url}")
    print(f"知识库 ID: {args.kb_id}")
    print(f"批次大小: {args.batch_size}")
    print("=" * 60)
    
    try:
        with open(args.input_file, 'r', encoding='utf-8') as f:
            qa_list = json.load(f)
        
        print(f"\n✅ 成功读取 {len(qa_list)} 条记录")
        
        if args.start_index > 0:
            print(f"⚠️  从第 {args.start_index + 1} 条开始导入（断点续传）")
        
        importer = QABatchImporter(
            api_url=args.api_url,
            token=args.token,
            knowledge_base_id=args.kb_id,
            batch_size=args.batch_size
        )
        
        start_time = time.time()
        
        importer.import_batch(qa_list, start_index=args.start_index)
        
        elapsed_time = time.time() - start_time
        
        importer.print_stats()
        importer.save_failed_records(args.failed_log)
        
        print(f"\n总耗时: {elapsed_time:.2f} 秒")
        print(f"平均速度: {len(qa_list) / elapsed_time:.2f} 条/秒")
        
        if importer.stats['failed'] > 0:
            print(f"\n⚠️  部分记录导入失败，请检查 {args.failed_log}")
            print(f"可使用 --start-index 参数重试失败的记录")
            sys.exit(1)
        else:
            print("\n✅ 全部导入成功！")
            sys.exit(0)
        
    except FileNotFoundError:
        print(f"❌ 错误: 文件 '{args.input_file}' 不存在")
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"❌ 错误: JSON 格式无效 - {e}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n\n⚠️  用户中断导入")
        print(f"已导入 {importer.stats['success']} 条记录")
        print(f"可使用 --start-index {importer.stats['success']} 继续导入")
        sys.exit(130)
    except Exception as e:
        print(f"❌ 错误: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
