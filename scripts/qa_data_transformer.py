#!/usr/bin/env python3
"""
历史客服问答数据转换脚本

功能：
1. 读取 JSON 格式的历史问答数据
2. 清洗 HTML 标签和特殊字符
3. 采用整体对话式方案组织数据
4. 将 category 字段存入 metadata
5. 输出适合导入知识库的格式
"""

import json
import re
import sys
from html import unescape
from typing import List, Dict, Any
from bs4 import BeautifulSoup


class QADataTransformer:
    """问答数据转换器"""
    
    def __init__(self):
        self.stats = {
            'total': 0,
            'success': 0,
            'failed': 0,
            'skipped': 0
        }
    
    def clean_html_content(self, html_text: str) -> str:
        """
        清洗 HTML 内容
        
        Args:
            html_text: 包含 HTML 标签的文本
            
        Returns:
            清洗后的纯文本
        """
        if not html_text:
            return ""
        
        try:
            soup = BeautifulSoup(html_text, 'html.parser')
            text = soup.get_text(separator=' ', strip=True)
            text = unescape(text)
            text = re.sub(r'\s+', ' ', text)
            text = re.sub(r'\n{3,}', '\n\n', text)
            
            return text.strip()
        except Exception as e:
            print(f"警告: HTML 清洗失败，使用简单清洗: {e}")
            text = re.sub(r'<[^>]+>', ' ', html_text)
            text = unescape(text)
            text = re.sub(r'\s+', ' ', text).strip()
            return text
    
    def build_conversational_passage(self, qa: Dict[str, Any]) -> str:
        """
        构建整体对话式段落
        
        将完整的问答对话组织成一个连贯的文本段落，保留完整上下文。
        
        Args:
            qa: 单条问答数据
            
        Returns:
            格式化的对话文本
        """
        lines = []
        
        lines.append("=" * 60)
        lines.append(f"问题标题: {self.clean_html_content(qa.get('title', ''))}")
        lines.append("")
        
        description = self.clean_html_content(qa.get('description', ''))
        if description:
            lines.append(f"问题描述: {description}")
            lines.append("")
        
        category = qa.get('category', '未分类')
        lines.append(f"分类: {category}")
        lines.append("")
        
        lines.append("对话记录:")
        lines.append("-" * 60)
        
        replies = qa.get('replies', [])
        for i, reply in enumerate(replies, 1):
            owner = reply.get('owner', 'unknown')
            owner_label = "客户" if owner == "customer" else "客服"
            
            content = self.clean_html_content(reply.get('content', ''))
            if not content:
                continue
            
            lines.append(f"{i}. [{owner_label}] {content}")
            lines.append("")
        
        lines.append("=" * 60)
        
        return "\n".join(lines)
    
    def extract_metadata(self, qa: Dict[str, Any]) -> Dict[str, Any]:
        """
        提取并构建 metadata
        
        Args:
            qa: 单条问答数据
            
        Returns:
            metadata 字典
        """
        from datetime import datetime
        
        metadata = {
            "qa_id": str(qa.get('id', '')),
            "category": qa.get('category', '未分类'),
            "source": "historical_qa",
            "import_date": datetime.now().strftime("%Y-%m-%d"),
            "reply_count": len(qa.get('replies', []))
        }
        
        return metadata
    
    def validate_qa(self, qa: Dict[str, Any]) -> bool:
        """
        验证问答数据是否有效
        
        Args:
            qa: 单条问答数据
            
        Returns:
            是否有效
        """
        if not qa.get('title') and not qa.get('description'):
            return False
        
        replies = qa.get('replies', [])
        if not replies:
            return False
        
        has_valid_content = False
        for reply in replies:
            content = self.clean_html_content(reply.get('content', ''))
            if content:
                has_valid_content = True
                break
        
        return has_valid_content
    
    def transform_single_qa(self, qa: Dict[str, Any]) -> Dict[str, Any]:
        """
        转换单条问答数据
        
        Args:
            qa: 原始问答数据
            
        Returns:
            转换后的数据，包含 passage 和 metadata
        """
        if not self.validate_qa(qa):
            raise ValueError("问答数据无效或内容为空")
        
        passage = self.build_conversational_passage(qa)
        metadata = self.extract_metadata(qa)
        title = self.clean_html_content(qa.get('title', ''))
        description = self.clean_html_content(qa.get('description', ''))
        
        return {
            "title": title,
            "description": description,
            "passage": passage,
            "metadata": metadata
        }
    
    def transform_batch(self, qa_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        批量转换问答数据
        
        Args:
            qa_list: 问答数据列表
            
        Returns:
            转换后的数据列表
        """
        self.stats['total'] = len(qa_list)
        transformed_list = []
        
        for i, qa in enumerate(qa_list, 1):
            try:
                transformed = self.transform_single_qa(qa)
                transformed_list.append(transformed)
                self.stats['success'] += 1
                
                if i % 100 == 0:
                    print(f"已处理 {i}/{self.stats['total']} 条记录")
                    
            except ValueError as e:
                self.stats['skipped'] += 1
                print(f"跳过第 {i} 条记录 (ID: {qa.get('id', 'unknown')}): {e}")
            except Exception as e:
                self.stats['failed'] += 1
                print(f"处理第 {i} 条记录失败 (ID: {qa.get('id', 'unknown')}): {e}")
        
        return transformed_list
    
    def print_stats(self):
        """打印统计信息"""
        print("\n" + "=" * 60)
        print("数据转换统计:")
        print(f"  总计: {self.stats['total']} 条")
        print(f"  成功: {self.stats['success']} 条")
        print(f"  跳过: {self.stats['skipped']} 条")
        print(f"  失败: {self.stats['failed']} 条")
        print("=" * 60)


def main():
    """主函数"""
    if len(sys.argv) < 3:
        print("用法: python qa_data_transformer.py <输入JSON文件> <输出JSON文件>")
        print("\n示例:")
        print("  python qa_data_transformer.py raw_qa_data.json transformed_qa_data.json")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    print(f"正在读取文件: {input_file}")
    
    try:
        with open(input_file, 'r', encoding='utf-8') as f:
            qa_list = json.load(f)
        
        print(f"成功读取 {len(qa_list)} 条问答记录\n")
        
        transformer = QADataTransformer()
        print("开始转换数据...")
        
        transformed_list = transformer.transform_batch(qa_list)
        
        transformer.print_stats()
        
        print(f"\n正在保存到文件: {output_file}")
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(transformed_list, f, ensure_ascii=False, indent=2)
        
        print(f"✅ 转换完成！已保存 {len(transformed_list)} 条记录")
        
    except FileNotFoundError:
        print(f"❌ 错误: 文件 '{input_file}' 不存在")
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"❌ 错误: JSON 格式无效 - {e}")
        sys.exit(1)
    except Exception as e:
        print(f"❌ 错误: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
