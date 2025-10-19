#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
小说爬虫：支持混合章节格式 + 自定义规则 + 封面 + 调试HTML + 屏蔽重复置顶章
作者：根据用户需求持续优化
"""

import os
import re
import time
import traceback
import requests
from urllib.parse import urljoin
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from bs4 import BeautifulSoup
from ebooklib import epub

# === 配置 ===
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
NOVEL_DIR = os.path.join(SCRIPT_DIR, "novel")
os.makedirs(NOVEL_DIR, exist_ok=True)

# Chromium 路径（Linux 常见路径）
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

def load_rules_from_file():
    rules_path = os.path.join(SCRIPT_DIR, "rules.txt")
    if os.path.exists(rules_path):
        with open(rules_path, "r", encoding="utf-8") as f:
            rules = [line.strip() for line in f if line.strip()]
        print(f"✅ 从 rules.txt 加载 {len(rules)} 条规则")
        return rules
    else:
        print("⚠️ rules.txt 不存在，跳过文件加载")
        return []

def get_user_rules():
    use_custom = input("是否使用自定义章节匹配规则？(y/n，默认 n): ").strip().lower()
    if use_custom == 'y':
        method = input("输入方式：1=交互输入，2=从 rules.txt 加载 (默认 2): ").strip()
        if method == '1':
            rule = input("请输入章节标题正则表达式: ").strip()
            if rule:
                return [rule]
            else:
                print("未输入规则，使用默认规则")
        elif method == '2' or not method:
            rules = load_rules_from_file()
            if rules:
                return rules
            else:
                print("rules.txt 为空或不存在，使用默认规则")
    return [
        r'\d+\.\s*(第?\d+章?)',
        r'第[零一二三四五六七八九十百千]+章',
        r'第\d+章',
        r'^\d{3,}\s+.+',
        r'\d+\s*[-–—]\s*.*',
        r'Chapter\s*\d+',
        r'【[^】]*\s*\d+】',
        r'第[零一二三四五六七八九十百千万]+[回节篇卷]'
    ]

def extract_chapter_links(toc_url, rules):
    print("正在加载目录页...")
    driver.get(toc_url)
    
    # 自动滚动加载全部章节
    for _ in range(8):
        driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
        time.sleep(1.2)

    soup = BeautifulSoup(driver.page_source, 'lxml')
    all_a_tags = soup.find_all('a', href=True)
    compiled_rules = [re.compile(rule, re.IGNORECASE) for rule in rules]
    seen_hrefs = set()
    final_links = []

    for a in all_a_tags:
        href = a['href']
        if href.startswith('/'):
            full_href = urljoin(toc_url, href)
        elif not href.startswith(('http://', 'https://')):
            full_href = toc_url.rstrip('/') + '/' + href.lstrip('/')
        else:
            full_href = href

        if full_href in seen_hrefs:
            continue

        text = a.get_text(strip=True)
        if any(pattern.search(text) for pattern in compiled_rules):
            final_links.append((text, full_href))
            seen_hrefs.add(full_href)

    print(f"✅ 初始匹配 {len(final_links)} 章（按页面原始顺序）")

    # 屏蔽前 N 章（防置顶重复）
    skip_n = 0
    if len(final_links) > 10:
        try:
            skip_input = input("是否屏蔽目录页前 N 章（防置顶重复）？(直接回车=0): ").strip()
            skip_n = int(skip_input) if skip_input.isdigit() else 0
        except:
            skip_n = 0

    if skip_n > 0:
        print(f"⚠️ 屏蔽前 {skip_n} 章")
        final_links = final_links[skip_n:]

    print(f"📌 最终保留 {len(final_links)} 章")

    # 预览
    print("前5章预览:")
    for i, (title, _) in enumerate(final_links[:5]):
        print(f"  {i+1}. {title}")
    if len(final_links) > 10:
        print("后5章预览:")
        for i, (title, _) in enumerate(final_links[-5:], start=len(final_links)-4):
            print(f"  {i}. {title}")

    confirm = input("章节顺序和内容正确吗？(y/n，默认 y): ").strip().lower()
    if confirm in ('', 'y', 'yes'):
        return final_links
    else:
        print("❌ 用户否决")
        return []

def get_chapter_content(ch_url):
    driver.get(ch_url)
    time.sleep(1.2)
    page_source = driver.page_source
    soup = BeautifulSoup(page_source, 'lxml')

    debug_dir = os.path.join(NOVEL_DIR, "debug_pages")
    os.makedirs(debug_dir, exist_ok=True)
    safe_name = re.sub(r'[\\/:*?"<>|]', '_', ch_url.replace('https://', '').replace('http://', ''))[:100]
    with open(os.path.join(debug_dir, f"{safe_name}.html"), "w", encoding="utf-8") as f:
        f.write(page_source)

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

    for elem in soup.find_all(string=re.compile(r'.', re.DOTALL)):
        if elem.parent and elem.parent.name in ['h1', 'h2', 'h3', 'h4']:
            txt = elem.strip()
            if len(txt) > 5 and any(kw in txt for kw in ['第', '章', '回', '节', 'Chapter', 'Episode', '【']):
                parent = elem.parent
                parts = []
                for sibling in parent.next_siblings:
                    if hasattr(sibling, 'get_text'):
                        t = sibling.get_text(strip=True)
                        if len(t) > 10:
                            parts.append(str(sibling))
                if parts:
                    print("  ✅ 通过标题后兄弟节点提取成功")
                    return ''.join(parts)

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

            if 'cover_data' in locals() and cover_data is not None:
                cover_file_name = f"cover.{cover_ext}"
                book.set_cover(cover_file_name, cover_data)

        except Exception as e:
            print(f"⚠️ 封面加载失败: {e}")

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

        rules = get_user_rules()
        chapter_list = extract_chapter_links(TOC_URL, rules)
        if not chapter_list:
            print("❌ 未识别到有效章节，请检查规则或网页结构。")
            exit(1)

        driver.get(TOC_URL)
        time.sleep(1)
        try:
            default_title = driver.title.replace('目录', '').replace('小说', '').replace('最新章节', '').strip()
        except:
            default_title = "我的小说"

        print("\n--- EPUB 元数据设置（直接回车使用默认值） ---")
        custom_title = input(f"书名（默认: {default_title}）: ").strip() or default_title
        custom_author = input("作者（默认: 匿名）: ").strip() or "匿名"
        cover_input = input("封面（本地路径如 cover.jpg，或网络链接，留空则无封面）: ").strip()

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