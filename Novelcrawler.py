#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
小说爬虫：输入目录页 → 下载章节 → 生成带封面的 EPUB
支持自定义标题、作者、封面（本地或网络）
适配 Chromium + chromedriver（已在 PATH 中）
"""

import os
import re
import time
import traceback
import requests
from urllib.parse import urlparse
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from bs4 import BeautifulSoup
from ebooklib import epub

# === 配置 ===
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
NOVEL_DIR = os.path.join(SCRIPT_DIR, "novel")
os.makedirs(NOVEL_DIR, exist_ok=True)

# Chromium 路径（Linux 常见路径，可按需修改）
CHROMIUM_PATH = "/usr/bin/chromium"

chrome_options = Options()
chrome_options.binary_location = CHROMIUM_PATH
chrome_options.add_argument("--headless")
chrome_options.add_argument("--no-sandbox")
chrome_options.add_argument("--disable-dev-shm-usage")
chrome_options.add_argument("--disable-gpu")
chrome_options.add_argument("--disable-images")
chrome_options.add_argument("--log-level=3")

print("正在启动浏览器...")
driver = webdriver.Chrome(options=chrome_options)

def sanitize_filename(name):
    return re.sub(r'[\\/:*?"<>|]', '_', name).strip() or "novel"

def extract_chapter_links(toc_url):
    print("正在加载目录页...")
    driver.get(toc_url)
    time.sleep(2)
    soup = BeautifulSoup(driver.page_source, 'lxml')

    links = []
    for a in soup.find_all('a', href=True):
        text = a.get_text(strip=True)
        if re.search(r'第[零一二三四五六七八九十百\d]+章', text):
            href = a['href']
            if href.startswith('/'):
                from urllib.parse import urljoin
                href = urljoin(toc_url, href)
            elif not href.startswith(('http://', 'https://')):
                href = toc_url.rstrip('/') + '/' + href.lstrip('/')
            links.append((text, href))
    return links

def get_chapter_content(ch_url):
    driver.get(ch_url)
    time.sleep(1.2)
    page_source = driver.page_source
    soup = BeautifulSoup(page_source, 'lxml')

    # 保存原始页面用于调试
    debug_dir = os.path.join(NOVEL_DIR, "debug_pages")
    os.makedirs(debug_dir, exist_ok=True)
    safe_name = re.sub(r'[\\/:*?"<>|]', '_', ch_url.replace('https://', '').replace('http://', ''))[:100]
    with open(os.path.join(debug_dir, f"{safe_name}.html"), "w", encoding="utf-8") as f:
        f.write(page_source)

    # 策略1：常见正文容器
    selectors = [
        '#content', '.content', '#chapter-content', '.chapter-content',
        '.read-content', '#read-content', '.txt', '.text', '.article-content',
        'article', 'main', '.post-content', '.entry-content', '.bookcontent'
    ]
    for sel in selectors:
        elem = soup.select_one(sel)
        if elem:
            print(f"  ✅ 使用选择器 '{sel}' 提取成功")
            return str(elem)

    # 策略2：从“第X章”标题往后找兄弟节点
    for elem in soup.find_all(string=re.compile(r'第[零一二三四五六七八九十百\d]+章')):
        title_elem = elem.parent
        if title_elem and title_elem.parent:
            parts = []
            for sibling in title_elem.parent.next_siblings:
                if hasattr(sibling, 'name') and sibling.name in ['p', 'div']:
                    txt = sibling.get_text(strip=True)
                    if len(txt) > 10:
                        parts.append(str(sibling))
            if parts:
                print("  ✅ 通过标题后兄弟节点提取成功")
                return ''.join(parts)

    # 策略3：退化到所有长段落
    paragraphs = soup.find_all('p')
    long_ps = [str(p) for p in paragraphs if len(p.get_text(strip=True)) > 20]
    if len(long_ps) > 2:
        print("  ⚠️ 退化：使用 <p> 标签组合")
        return ''.join(long_ps)

    print("  ❌ 正文提取失败！原始页面已保存到 debug_pages/")
    return "<p>正文提取失败，请查看 debug_pages/ 中的 HTML 文件。</p>"

def create_epub(title, author, cover_path_or_url, chapters, output_dir):
    safe_title = sanitize_filename(title or "小说")
    book = epub.EpubBook()
    book.set_identifier(f'novel_{abs(hash(safe_title)) % (10**8)}')
    book.set_title(title or "小说")
    book.set_language('zh')
    book.add_author(author or "匿名")

    # 处理封面
    if cover_path_or_url:
        try:
            if cover_path_or_url.startswith(('http://', 'https://')):
                print("正在下载封面...")
                resp = requests.get(cover_path_or_url, timeout=10)
                resp.raise_for_status()
                cover_data = resp.content
                content_type = resp.headers.get('content-type', 'image/jpeg')
                if 'png' in content_type:
                    cover_ext = "png"
                elif 'gif' in content_type:
                    cover_ext = "gif"
                else:
                    cover_ext = "jpg"
            else:
                if not os.path.exists(cover_path_or_url):
                    print(f"⚠️ 封面文件不存在: {cover_path_or_url}，跳过封面")
                    cover_data = None
                else:
                    with open(cover_path_or_url, "rb") as f:
                        cover_data = f.read()
                    _, ext = os.path.splitext(cover_path_or_url)
                    cover_ext = ext.lower().lstrip('.')

            if cover_data:
                cover_file_name = f"cover.{cover_ext}"
                book.set_cover(cover_file_name, cover_data)

        except Exception as e:
            print(f"⚠️ 封面加载失败: {e}")

    # 添加章节
    spine = ['nav']
    toc = []
    for i, (ch_title, ch_html) in enumerate(chapters):
        chapter = epub.EpubHtml(
            title=ch_title,
            file_name=f'chap_{i+1:04}.xhtml',
            lang='zh'
        )
        chapter.content = f'<h1>{ch_title}</h1>' + (ch_html or "<p>空内容</p>")
        book.add_item(chapter)
        toc.append(chapter)
        spine.append(chapter)

    book.toc = tuple(toc)
    book.add_item(epub.EpubNcx())
    book.add_item(epub.EpubNav())
    book.spine = spine

    epub_path = os.path.join(output_dir, f"{safe_title}.epub")
    epub.write_epub(epub_path, book)
    return epub_path

# === 主程序 ===
if __name__ == '__main__':
    try:
        TOC_URL = input("请输入小说目录页完整链接（以 http:// 或 https:// 开头）: ").strip()
        if not TOC_URL.startswith(('http://', 'https://')):
            print("❌ 链接格式错误！")
            exit(1)

        chapter_list = extract_chapter_links(TOC_URL)
        print(f"✅ 共识别到 {len(chapter_list)} 章")

        if not chapter_list:
            print("⚠️ 没有找到任何章节，请确认页面包含“第X章”字样。")
            exit(1)

        # 获取默认书名
        driver.get(TOC_URL)
        time.sleep(1)
        try:
            default_title = driver.title.replace('目录', '').replace('小说', '').replace('最新章节', '').strip()
        except:
            default_title = "我的小说"

        # 用户自定义元数据
        print("\n--- EPUB 元数据设置（直接回车使用默认值） ---")
        custom_title = input(f"书名（默认: {default_title}）: ").strip() or default_title
        custom_author = input("作者（默认: 匿名）: ").strip() or "匿名"
        cover_input = input("封面（本地路径如 cover.jpg，或网络链接，留空则无封面）: ").strip()

        # 下载章节
        chapters_content = []
        for i, (ch_title, ch_url) in enumerate(chapter_list):
            print(f"[{i+1}/{len(chapter_list)}] 正在下载：{ch_title}")
            try:
                content = get_chapter_content(ch_url)
                chapters_content.append((ch_title, content))
                print(f"    → 提取到 {len(content)} 字符")
            except Exception as e:
                print(f"    ❌ 错误: {e}")
                traceback.print_exc()
                chapters_content.append((ch_title, "<p>加载异常</p>"))

        # 生成 EPUB
        print("正在生成 EPUB...")
        epub_file = create_epub(
            title=custom_title,
            author=custom_author,
            cover_path_or_url=cover_input,
            chapters=chapters_content,
            output_dir=NOVEL_DIR
        )
        print(f"🎉 完成！EPUB 已保存至：{epub_file}")
        print(f"📄 调试 HTML 在：{os.path.join(NOVEL_DIR, 'debug_pages')}")

    except KeyboardInterrupt:
        print("\n用户中断。")
    except Exception as e:
        print(f"❌ 程序崩溃: {e}")
        traceback.print_exc()
    finally:
        print("正在关闭浏览器...")
        driver.quit()