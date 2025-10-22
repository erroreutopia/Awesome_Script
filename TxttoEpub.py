import os
import sys
import re
import mimetypes
import chardet
from ebooklib import epub

# 更宽松的章节正则（支持中英文常见格式）
CHAPTER_PATTERNS = [
    r'^第[零一二三四五六七八九十百千\d]+[章回节卷篇]\s*.+',
    r'^Chapter\s+\d+[:\s\-].*',
    r'^CHAPTER\s+\d+[:\s\-].*',
    r'^##\s*.+',
    r'^={3,}\s*.+\s*={3,}$',
    r'^\d+\s*[\.\-\s].+',  # 如 "1. 引言"
]

def is_chapter_title(line):
    line = line.strip()
    if not line:
        return False
    for pattern in CHAPTER_PATTERNS:
        if re.match(pattern, line, re.IGNORECASE):
            return True
    return False

def read_text_file(filepath):
    with open(filepath, 'rb') as f:
        raw_data = f.read()
    result = chardet.detect(raw_data)
    detected_encoding = result['encoding']
    confidence = result['confidence']
    print(f"chardet 检测编码: {detected_encoding} (置信度: {confidence:.2f})")

    encodings_to_try = ['gbk', 'utf-8', 'gb2312', 'latin1'] if detected_encoding and ('gb' in detected_encoding.lower()) else ['utf-8', 'gbk', 'gb2312']
    for enc in encodings_to_try:
        try:
            return raw_data.decode(enc)
        except UnicodeDecodeError:
            continue
    raise UnicodeDecodeError("无法解码文件")

def split_into_chapters(lines):
    chapters = []
    current_title = "未命名章节"
    current_content = []
    in_content = False

    for line in lines:
        stripped = line.strip()
        if is_chapter_title(line):
            # 保存上一章
            if current_content or chapters:  # 避免首章为空
                chapters.append((current_title, current_content))
            # 开始新章节
            current_title = stripped
            current_content = []
            in_content = True
        else:
            if in_content or not chapters:  # 首章前的内容归入第一章
                current_content.append(line)
            # 否则忽略章前空白（如文件开头的说明）

    # 添加最后一章
    if current_content or not chapters:
        chapters.append((current_title, current_content))
    
    # 如果整本书没识别到章节，合并为一章
    if len(chapters) == 1 and not is_chapter_title(chapters[0][0]):
        chapters = [("正文", lines)]
    
    return chapters

def txt_to_epub(txt_path, epub_path=None, title="Untitled", author="Unknown", cover_path=None):
    if not os.path.isfile(txt_path):
        print(f"错误：输入文件 {txt_path} 不存在。")
        return

    output_path = epub_path or os.path.splitext(txt_path)[0] + '.epub'
    if os.path.abspath(output_path) == os.path.abspath(txt_path):
        print("❌ 错误：输出文件不能与输入文件同名！")
        return

    try:
        content = read_text_file(txt_path)
    except Exception as e:
        print(f"读取失败: {e}")
        return

    lines = content.splitlines()
    chapters = split_into_chapters(lines)
    print(f"📖 识别到 {len(chapters)} 个章节")

    book = epub.EpubBook()
    book.set_identifier('id123456')
    book.set_title(title)
    book.set_language('zh')
    book.add_author(author)

    # 🖼️ 封面
    if cover_path and os.path.isfile(cover_path):
        mime_type, _ = mimetypes.guess_type(cover_path)
        if mime_type and mime_type.startswith('image/'):
            with open(cover_path, 'rb') as f:
                cover_image = f.read()
            book.set_cover("cover.jpg", cover_image)
            print(f"✅ 已添加封面: {cover_path}")

    # 📚 创建章节
    epub_chapters = []
    toc_entries = []
    for i, (chap_title, chap_lines) in enumerate(chapters):
        html_lines = []
        for line in chap_lines:
            if line.strip() == '':
                html_lines.append('<p>&nbsp;</p>')
            else:
                html_lines.append(f'<p>{line}</p>')
        html_content = ''.join(html_lines)

        file_name = f'chapter_{i+1:03}.xhtml'
        chapter = epub.EpubHtml(title=chap_title, file_name=file_name, lang='zh')
        chapter.content = f'<h1>{chap_title}</h1>' + html_content
        book.add_item(chapter)
        epub_chapters.append(chapter)
        toc_entries.append(epub.Link(file_name, chap_title, f'chap_{i+1}'))

    # 🧭 目录和结构
    book.toc = tuple(toc_entries)
    book.add_item(epub.EpubNcx())
    book.add_item(epub.EpubNav())
    book.spine = ['nav'] + epub_chapters

    epub.write_epub(output_path, book)
    print(f"✅ 已生成 EPUB 文件：{output_path}")

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("用法: python txt_to_epub.py <input.txt> [output.epub] [书名] [作者] [cover.jpg]")
        sys.exit(1)

    txt_file = sys.argv[1]
    epub_file = sys.argv[2] if len(sys.argv) > 2 else None
    title = sys.argv[3] if len(sys.argv) > 3 else "Untitled"
    author = sys.argv[4] if len(sys.argv) > 4 else "Unknown"
    cover = sys.argv[5] if len(sys.argv) > 5 else None

    txt_to_epub(txt_file, epub_file, title, author, cover)