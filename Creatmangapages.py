import os
from pathlib import Path
import re
import json # 需要导入 json 来序列化全局目录信息

# 全局变量，用于存储整个项目的目录结构信息
# 键是相对于根目录的路径 ('.' 代表根目录)，值是包含 'name', 'tags', 'clean_name' 等信息的字典
all_directories_info = {}

def collect_all_subdirs(root_path):
    """
    递归收集所有子目录信息，用于全局搜索。
    填充全局字典 all_directories_info。
    """
    global all_directories_info
    all_directories_info = {} # 重置字典

    def _scan_dir(current_path, relative_base=Path('.')):
        try:
            items = list(current_path.iterdir())
        except PermissionError:
            print(f"警告: 无法访问目录 '{current_path}' (权限不足).")
            return

        subdirs = []

        for item in items:
            if item.is_dir() and not item.name.startswith('.dotfile_background'):
                subdirs.append(item.name)

        # 保存当前目录信息
        relative_path_str = relative_base.as_posix() # 转换为字符串路径
        display_name = relative_path_str if relative_path_str != '.' else "根目录"

        # 提取标签
        tags = re.findall(r'\[\[([^\]]+)\]\]', display_name)
        clean_name = re.sub(r'\[\[([^\]]+)\]\]', '', display_name).strip()
        if clean_name.startswith('.'):
            clean_name = f"·{clean_name[1:]}"
        elif not clean_name:
            clean_name = display_name

        all_directories_info[relative_path_str] = {
            'name': display_name,
            'clean_name': clean_name,
            'full_path': relative_path_str, # 存储完整相对路径
            'tags': tags
        }

        # 递归扫描子目录
        for subdir in subdirs:
            subdir_path = current_path / subdir
            new_relative_base = relative_base / subdir
            _scan_dir(subdir_path, new_relative_base)

    _scan_dir(root_path)

def generate_index_html(directory_path):
    """
    为给定目录生成 index.html 文件。
    包含该目录下的所有支持的图片和子目录链接。
    递归处理所有子目录。
    """
    directory_path = Path(directory_path).resolve()
    root_path = Path.cwd().resolve()
    # 支持的图片扩展名
    image_extensions = {'.jpg', '.jpeg', '.png', '.webp', '.gif', '.bmp', '.tiff'}
    # 获取目录内容
    try:
        items = list(directory_path.iterdir())
    except PermissionError:
        print(f"警告: 无法访问目录 '{directory_path}' (权限不足).")
        return
    # 分离图片文件和子目录
    image_files = []
    subdirectories = []
    for item in items:
        if item.is_file() and item.suffix.lower() in image_extensions:
            # 隐藏点文件背景图片
            if not item.name.startswith('.dotfile_background'):
                image_files.append(item.name)
        elif item.is_dir(): # 不再忽略隐藏目录，但不在面包屑中显示
            subdirectories.append(item.name)
    # 对文件和目录进行排序
    image_files.sort(key=str.lower)
    subdirectories.sort(key=str.lower)
    # 提取目录标签
    dir_tags = {}
    visible_subdirs = []
    for dir_name in subdirectories:
        # 过滤掉点文件背景目录
        if not dir_name.startswith('.dotfile_background'):
            # 提取标签 [[tag]]
            tags = re.findall(r'\[\[([^\]]+)\]\]', dir_name)
            if tags:
                dir_tags[dir_name] = tags
            visible_subdirs.append(dir_name)
    # 生成 HTML 内容
    relative_path_from_root = directory_path.relative_to(root_path)
    page_title = relative_path_from_root.as_posix() if relative_path_from_root != Path('.') else '根目录'
    # 计算返回根目录的路径（用于背景图片路径）
    depth = len(relative_path_from_root.parts) if relative_path_from_root != Path('.') else 0
    back_to_root_path = "../" * depth
    # 构建图片列表的JavaScript数组
    js_image_list = "[" + ", ".join([f'"{img}"' for img in image_files]) + "]"
    # 构建子目录列表的JavaScript数组（用于搜索，排除点文件背景）
    filtered_subdirs = [dir for dir in subdirectories if not dir.startswith('.')]
    js_subdirs_list = "[" + ", ".join([f'"{dir}"' for dir in filtered_subdirs]) + "]"
    # 构建目录标签的JavaScript对象
    js_dir_tags = "{"
    for dir_name, tags in dir_tags.items():
        js_dir_tags += f'"{dir_name}": ["' + '", "'.join(tags) + '"], '
    if js_dir_tags.endswith(', '):
        js_dir_tags = js_dir_tags[:-2]
    js_dir_tags += "}"
    
    # 只在根目录收集一次全局信息
    if directory_path == root_path:
        print("正在收集全局目录信息用于搜索...")
        collect_all_subdirs(root_path)
    
    # 将全局目录信息序列化为 JSON 字符串，供 JavaScript 使用
    # ensure_ascii=False 保持中文可读性，separators 使输出更紧凑
    js_all_dirs = json.dumps(all_directories_info, ensure_ascii=False, separators=(',', ':'))

    html_content = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>图片库 - {page_title}</title>
    <style id="theme-styles">
        /* 默认主题 (浅色) */
        :root {{
            --bg-color: #fdf6f0;
            --text-color: #5a3e36;
            --header-bg: rgba(255, 255, 255, 0.9);
            --card-bg: rgba(255, 255, 255, 0.85);
            --border-color: #d9c7b0;
            --shadow-color: rgba(180, 150, 130, 0.3);
            --button-bg: #f8f0e5;
            --button-hover: #f0e0d0;
            --primary-color: #ff9a8b;
            --primary-hover: #ff6b6b;
            --nav-button-bg: rgba(255, 255, 255, 0.9);
            --nav-button-hover: rgba(255, 255, 255, 1);
            --zoom-button-bg: rgba(255, 255, 255, 0.9);
            --zoom-button-hover: rgba(255, 255, 255, 1);
            --page-number-bg: rgba(0, 0, 0, 0.5);
            --page-number-color: #fff;
            --accent-color: #ffb6c1;
            --accent-hover: #ff9aa2;
            --search-bg: rgba(255, 255, 255, 0.9);
            --tag-bg: #ffe5e5;
            --tag-color: #d63384;
        }}
        body.dark-theme {{
            --bg-color: #2b2b3b;
            --text-color: #e0d6eb;
            --header-bg: rgba(50, 50, 65, 0.9);
            --card-bg: rgba(50, 50, 65, 0.85);
            --border-color: #6a5a8c;
            --shadow-color: rgba(0, 0, 0, 0.5);
            --button-bg: #3a3a4a;
            --button-hover: #4a4a5a;
            --primary-color: #ff7bac;
            --primary-hover: #ff5299;
            --nav-button-bg: rgba(50, 50, 65, 0.9);
            --nav-button-hover: rgba(70, 70, 90, 1);
            --zoom-button-bg: rgba(50, 50, 65, 0.9);
            --zoom-button-hover: rgba(70, 70, 90, 1);
            --page-number-bg: rgba(255, 255, 255, 0.2);
            --page-number-color: #fff;
            --accent-color: #c29fbf;
            --accent-hover: #a97fa7;
            --search-bg: rgba(50, 50, 65, 0.9);
            --tag-bg: #5a3e50;
            --tag-color: #ff7bac;
        }}
        body {{
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 0;
            background-color: var(--bg-color);
            color: var(--text-color);
            transition: background-color 0.3s, color 0.3s;
            position: relative;
            min-height: 100vh;
        }}
        /* 背景图片 */
        body.bg-enabled {{
            background-size: cover;
            background-position: center;
            background-repeat: no-repeat;
            background-attachment: fixed;
        }}
        body.bg-blur {{
            backdrop-filter: blur(5px);
        }}
        .container {{
            max-width: 100%;
            margin: 0 auto;
            padding: 20px;
        }}
        header {{
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: nowrap; /* 修改: 禁止换行 */
            padding: 15px 20px;
            background-color: var(--header-bg);
            border-bottom: 3px solid var(--primary-color);
            margin-bottom: 20px;
            position: sticky;
            top: 0;
            z-index: 100;
            border-radius: 0 0 10px 10px;
            box-shadow: 0 4px 12px var(--shadow-color);
        }}
        h1 {{
            margin: 0;
            color: var(--primary-color);
            font-size: 1.8em;
            flex: 1 1 auto; /* 新增: 弹性布局 */
            min-width: 0; /* 新增: 允许内容溢出容器 */
            overflow-x: auto; /* 新增: 水平滚动 */
            white-space: nowrap; /* 新增: 禁止换行 */
            padding-bottom: 5px; /* 新增: 为滚动条留出空间 */
            /* 添加滚动条样式 */
            scrollbar-width: thin;
            scrollbar-color: var(--border-color) transparent;
        }}
        h1::-webkit-scrollbar {{
            height: 8px; /* 新增: 水平滚动条高度 */
        }}
        h1::-webkit-scrollbar-track {{
            background: transparent;
        }}
        h1::-webkit-scrollbar-thumb {{
            background-color: var(--border-color);
            border-radius: 4px;
        }}
        .breadcrumb {{
            margin: 10px 0;
            font-size: 0.9em;
            overflow-x: auto; /* 新增: 水平滚动 */
            white-space: nowrap; /* 新增: 禁止换行 */
            padding-bottom: 5px; /* 新增: 为滚动条留出空间 */
            /* 添加滚动条样式 */
            scrollbar-width: thin;
            scrollbar-color: var(--border-color) transparent;
        }}
        .breadcrumb::-webkit-scrollbar {{
            height: 8px; /* 新增: 水平滚动条高度 */
        }}
        .breadcrumb::-webkit-scrollbar-track {{
            background: transparent;
        }}
        .breadcrumb::-webkit-scrollbar-thumb {{
            background-color: var(--border-color);
            border-radius: 4px;
        }}
        .breadcrumb a {{
            text-decoration: none;
            color: var(--primary-color);
            padding: 4px 8px;
            border-radius: 4px;
            transition: all 0.3s;
        }}
        .breadcrumb a:hover {{
            background-color: var(--button-hover);
            text-decoration: underline;
        }}
        .view-controls {{
            display: flex;
            gap: 10px;
            margin: 15px 0;
        }}
        .view-btn {{
            padding: 8px 15px;
            background-color: var(--button-bg);
            border: 2px solid var(--border-color);
            border-radius: 20px;
            cursor: pointer;
            transition: all 0.3s;
            font-weight: bold;
            box-shadow: 0 2px 5px var(--shadow-color);
        }}
        .view-btn.active {{
            background-color: var(--primary-color);
            color: white;
            border-color: var(--primary-hover);
        }}
        .view-btn:hover:not(.active) {{
            background-color: var(--button-hover);
            transform: translateY(-2px);
            box-shadow: 0 4px 8px var(--shadow-color);
        }}
        .theme-controls {{
            display: flex;
            gap: 10px;
            margin: 15px 0;
            align-items: center;
        }}
        .theme-toggle, .bg-toggle, .blur-toggle {{
            padding: 8px 15px;
            background-color: var(--button-bg);
            border: 2px solid var(--border-color);
            border-radius: 20px;
            cursor: pointer;
            transition: all 0.3s;
            font-weight: bold;
            box-shadow: 0 2px 5px var(--shadow-color);
        }}
        .theme-toggle:hover, .bg-toggle:hover, .blur-toggle:hover {{
            background-color: var(--button-hover);
            transform: translateY(-2px);
            box-shadow: 0 4px 8px var(--shadow-color);
        }}
        /* 搜索框 */
        .search-container {{
            position: relative;
            margin: 15px 0;
            width: 100%;
        }}
        .search-box {{
            width: 100%;
            padding: 10px 15px;
            border: 2px solid var(--border-color);
            border-radius: 20px;
            background-color: var(--search-bg);
            color: var(--text-color);
            font-size: 1em;
            box-shadow: 0 2px 5px var(--shadow-color);
            box-sizing: border-box;
        }}
        .search-box:focus {{
            outline: none;
            border-color: var(--primary-color);
            box-shadow: 0 0 8px var(--accent-color);
        }}
        .search-results {{
            position: absolute;
            top: 100%;
            left: 0;
            right: 0;
            background-color: var(--search-bg);
            border: 2px solid var(--border-color);
            border-top: none;
            border-radius: 0 0 20px 20px;
            max-height: 300px;
            overflow-y: auto;
            z-index: 200;
            box-shadow: 0 4px 12px var(--shadow-color);
            display: none;
        }}
        .search-result-item {{
            padding: 10px 15px;
            cursor: pointer;
            border-bottom: 1px solid var(--border-color);
        }}
        .search-result-item:last-child {{
            border-bottom: none;
        }}
        .search-result-item:hover {{
            background-color: var(--button-hover);
        }}
        .search-result-item a {{
            text-decoration: none;
            color: var(--text-color);
            display: block;
        }}
        .search-result-tags {{
            display: flex;
            flex-wrap: wrap;
            gap: 5px;
            margin-top: 5px;
        }}
        .search-result-tag {{
            background-color: var(--tag-bg);
            color: var(--tag-color);
            padding: 2px 8px;
            border-radius: 10px;
            font-size: 0.8em;
        }}
        /* 新增: 搜索结果中的路径和标签匹配标识 */
        .search-result-path {{
            font-size: 0.8em;
            color: var(--accent-color);
            margin-top: 3px;
        }}
        .search-result-tag-match-icon {{
            font-size: 0.8em;
            color: var(--tag-color);
            margin-right: 5px;
        }}
        /* 标签搜索 (已移除相关HTML/CSS) */
        /* 传统滚动视图 */
        #scroll-view {{
            display: block;
            width: 100%;
        }}
        .scroll-gallery {{
            display: flex;
            flex-direction: column;
            gap: 30px;
            width: 100%;
        }}
        .scroll-item {{
            background-color: var(--card-bg);
            border: 2px solid var(--border-color);
            border-radius: 15px;
            overflow: hidden;
            box-shadow: 0 4px 15px var(--shadow-color);
            transition: all 0.3s;
            width: 100%;
        }}
        .scroll-item:hover {{
            transform: translateY(-5px);
            box-shadow: 0 8px 20px var(--shadow-color);
            border-color: var(--accent-color);
        }}
        .scroll-item img {{
            width: 100%;
            height: auto;
            display: block;
            margin: 0 auto;
        }}
        .scroll-item .label {{
            padding: 12px;
            text-align: center;
            background-color: rgba(255, 255, 255, 0.7);
            border-top: 1px solid var(--border-color);
            font-weight: bold;
        }}
        body.dark-theme .scroll-item .label {{
            background-color: rgba(70, 70, 90, 0.7);
        }}
        /* 双页视图 */
        #book-view {{
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.9);
            z-index: 1000;
        }}
        .book-header {{
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            padding: 15px 20px;
            background-color: rgba(0, 0, 0, 0.7);
            display: flex;
            justify-content: space-between;
            align-items: center;
            z-index: 1001;
            border-bottom: 2px solid var(--primary-color);
        }}
        .book-title {{
            margin: 0;
            font-size: 1.2em;
            color: white;
        }}
        .book-controls {{
            display: flex;
            gap: 10px;
        }}
        .book-control-btn {{
            background-color: rgba(255, 255, 255, 0.2);
            color: white;
            border: 2px solid var(--border-color);
            border-radius: 20px;
            padding: 8px 12px;
            cursor: pointer;
            transition: all 0.3s;
            font-weight: bold;
            box-shadow: 0 2px 5px rgba(0,0,0,0.3);
        }}
        .book-control-btn:hover {{
            background-color: rgba(255, 255, 255, 0.3);
            transform: translateY(-2px);
            box-shadow: 0 4px 8px rgba(0,0,0,0.4);
        }}
        .pages-container {{
            position: absolute;
            top: 70px;
            left: 0;
            width: 100%;
            height: calc(100% - 70px);
            display: flex;
            justify-content: center;
            align-items: center;
            overflow: hidden;
        }}
        .pages-wrapper {{
            display: flex;
            justify-content: center;
            align-items: center;
            gap: 40px;
            width: 100%;
            height: 100%;
        }}
        .page {{
            flex: 1;
            text-align: center;
            max-width: 45%;
            height: 100%;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            position: relative;
        }}
        .page-content {{
            width: 100%;
            height: 100%;
            display: flex;
            justify-content: center;
            align-items: center;
            overflow: hidden;
            position: relative;
        }}
        .page img {{
            max-width: 100%;
            max-height: 100%;
            width: auto;
            height: auto;
            box-shadow: 0 8px 25px rgba(0,0,0,0.7);
            background-color: #fff;
            object-fit: contain;
            user-select: none;
            transform-origin: center center;
            border: 3px solid white;
            border-radius: 5px;
        }}
        .page-number {{
            position: absolute;
            bottom: 20px;
            font-size: 0.9em;
            color: var(--page-number-color);
            background-color: var(--page-number-bg);
            padding: 5px 10px;
            border-radius: 20px;
            font-weight: bold;
        }}
        .left-page-number {{
            left: 20px;
        }}
        .right-page-number {{
            right: 20px;
        }}
        .nav-button {{
            position: absolute;
            top: 50%;
            transform: translateY(-50%);
            background-color: var(--nav-button-bg);
            color: var(--primary-color);
            border: 2px solid var(--border-color);
            font-size: 2em;
            width: 70px;
            height: 70px;
            border-radius: 50%;
            cursor: pointer;
            display: flex;
            justify-content: center;
            align-items: center;
            transition: all 0.3s;
            user-select: none;
            z-index: 1001;
            box-shadow: 0 4px 15px rgba(0,0,0,0.5);
        }}
        .nav-button:hover {{
            background-color: var(--nav-button-hover);
            transform: translateY(-50%) scale(1.1);
            box-shadow: 0 6px 20px rgba(0,0,0,0.6);
            color: var(--primary-hover);
        }}
        .nav-button:disabled {{
            background-color: rgba(100, 100, 100, 0.3);
            cursor: not-allowed;
            transform: translateY(-50%);
            color: #888;
        }}
        #prev-button {{
            left: 20px;
        }}
        #next-button {{
            right: 20px;
        }}
        .zoom-controls {{
            position: absolute;
            bottom: 20px;
            left: 50%;
            transform: translateX(-50%);
            display: flex;
            gap: 10px;
            z-index: 1001;
        }}
        .zoom-btn {{
            width: 50px;
            height: 50px;
            border-radius: 50%;
            background-color: var(--zoom-button-bg);
            color: var(--primary-color);
            border: 2px solid var(--border-color);
            font-size: 1.5em;
            cursor: pointer;
            display: flex;
            justify-content: center;
            align-items: center;
            box-shadow: 0 4px 10px rgba(0,0,0,0.3);
        }}
        .zoom-btn:hover {{
            background-color: var(--zoom-button-hover);
            transform: scale(1.1);
            box-shadow: 0 6px 15px rgba(0,0,0,0.4);
            color: var(--primary-hover);
        }}
        .no-content {{
            text-align: center;
            padding: 40px;
            color: #999;
            font-style: italic;
        }}
        /* 目录网格 */
        .folders-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 20px;
            margin-top: 30px;
        }}
        .folder-item {{
            background-color: var(--card-bg);
            border: 2px solid var(--border-color);
            border-radius: 15px;
            overflow: hidden;
            box-shadow: 0 4px 15px var(--shadow-color);
            transition: all 0.3s;
        }}
        .folder-item:hover {{
            transform: translateY(-5px);
            box-shadow: 0 8px 20px var(--shadow-color);
            border-color: var(--accent-color);
        }}
        .folder-item a {{
            text-decoration: none;
            color: inherit;
            display: block;
        }}
        .folder-icon {{
            width: 100%;
            height: 150px;
            display: flex;
            align-items: center;
            justify-content: center;
            background-color: rgba(255, 255, 255, 0.7);
            font-size: 4em;
            color: var(--text-color);
        }}
        body.dark-theme .folder-icon {{
            background-color: rgba(70, 70, 90, 0.7);
        }}
        .folder-label {{
            padding: 12px;
            text-align: center;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            font-weight: bold;
        }}
        .folder-tags {{
            display: flex;
            flex-wrap: wrap;
            gap: 5px;
            padding: 0 12px 12px 12px;
        }}
        .folder-tag {{
            background-color: var(--tag-bg);
            color: var(--tag-color);
            padding: 2px 8px;
            border-radius: 10px;
            font-size: 0.8em;
        }}
        /* 移动端优化 */
        @media (max-width: 768px) {{
            .page {{
                max-width: 95%;
            }}
            .pages-wrapper {{
                gap: 10px;
            }}
            .nav-button {{
                width: 50px;
                height: 50px;
                font-size: 1.5em;
            }}
            #prev-button {{
                left: 10px;
            }}
            #next-button {{
                right: 10px;
            }}
            .zoom-controls {{
                bottom: 10px;
            }}
            .zoom-btn {{
                width: 40px;
                height: 40px;
                font-size: 1.1em;
            }}
            .book-header {{
                padding: 10px 15px;
            }}
            .book-title {{
                font-size: 1em;
            }}
            .book-control-btn {{
                padding: 6px 10px;
                font-size: 0.9em;
            }}
            header {{
                flex-direction: column;
                text-align: center;
            }}
            .controls {{
                width: 100%;
                display: flex;
                justify-content: center;
                flex-wrap: wrap;
            }}
            /* .tag-search-container 已移除 */
        }}
        /* 超小屏幕优化 */
        @media (max-width: 480px) {{
            .nav-button {{
                width: 40px;
                height: 40px;
                font-size: 1.2em;
            }}
            #prev-button {{
                left: 5px;
            }}
            #next-button {{
                right: 5px;
            }}
            .zoom-btn {{
                width: 35px;
                height: 35px;
                font-size: 1em;
            }}
            .page-number {{
                font-size: 0.8em;
                padding: 3px 8px;
            }}
            .left-page-number {{
                left: 10px;
            }}
            .right-page-number {{
                right: 10px;
            }}
        }}
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>图片库 - {page_title}</h1>
            <div class="controls">
                <div class="view-controls">
                    <button id="scroll-view-btn" class="view-btn active">传统滚动</button>
                    <button id="book-view-btn" class="view-btn">双页漫画</button>
                </div>
                <div class="theme-controls">
                    <button id="theme-toggle" class="theme-toggle">🌓 切换主题</button>
                    <button id="bg-toggle" class="bg-toggle">🖼️ 背景</button>
                    <button id="blur-toggle" class="blur-toggle">💧 模糊</button>
                </div>
            </div>
        </header>
        <div class="breadcrumb">
"""
    # 添加面包屑导航 (使用绝对路径避免循环)
    if directory_path != root_path:
        # 计算返回根目录需要的 "../" 层数
        depth = len(relative_path_from_root.parts)
        back_to_root = "../" * depth
        html_content += f'<a href="{back_to_root}">🏠 根目录</a>'
        # 添加中间路径
        path_so_far = Path('.')
        for i, part in enumerate(relative_path_from_root.parts):
            path_so_far /= part
            # 计算从当前目录返回到这个part需要的 "../" 层数
            up_levels = len(relative_path_from_root.parts) - i - 1
            href = "../" * up_levels if up_levels > 0 else "./"
            html_content += f' / <a href="{href}">{part}</a>'
    else:
        html_content += '<span>🏠 根目录</span>'
    html_content += """
        </div>
        <!-- 合并后的全局搜索 -->
        <div class="search-container">
            <input type="text" id="search-box" class="search-box" placeholder="🔍 全局搜索目录或标签...">
            <div id="search-results" class="search-results"></div>
        </div>
        <!-- 传统滚动视图 -->
        <div id="scroll-view">
"""
    # 添加图片项 (传统滚动视图)
    if image_files:
        html_content += '            <div class="scroll-gallery">\n'
        for img_name in image_files:
            html_content += f'''                <div class="scroll-item">
                    <img src="{img_name}" alt="{img_name}">
                    <div class="label">{img_name}</div>
                </div>
'''
        html_content += '            </div>\n'
    else:
        html_content += '            <div class="no-content">此目录中没有图片。</div>\n'
    html_content += """
        </div>
        <!-- 子目录 -->
        <h2>📁 子目录</h2>
"""
    # 添加子目录项 (排除点文件背景)
    if visible_subdirs:
        html_content += '        <div class="folders-grid">\n'
        for dir_name in visible_subdirs:
            # 隐藏目录使用特殊标记
            display_name = re.sub(r'\[\[([^\]]+)\]\]', '', dir_name).strip()  # 移除标签
            if display_name.startswith('.'):
                display_name = f"·{display_name[1:]}"  # 隐藏目录前加点
            elif not display_name:
                display_name = dir_name  # 如果移除标签后为空，显示原名
            # 获取目录标签
            tags_html = ""
            if dir_name in dir_tags:
                for tag in dir_tags[dir_name]:
                    tags_html += f'<span class="folder-tag">{tag}</span>'
            html_content += f'''            <div class="folder-item">
                <a href="{dir_name}/">
                    <div class="folder-icon">📁</div>
                    <div class="folder-label">{display_name}</div>
                    {f'<div class="folder-tags">{tags_html}</div>' if tags_html else ''}
                </a>
            </div>
'''
        html_content += '        </div>\n'
    else:
        html_content += '        <div class="no-content">此目录中没有子目录。</div>\n'
    html_content += """
    </div>
    <!-- 双页漫画视图 (全屏模式) -->
    <div id="book-view">
        <div class="book-header">
            <h2 class="book-title">📖 双页漫画视图</h2>
            <div class="book-controls">
                <button id="exit-book-view" class="book-control-btn">🚪 退出</button>
            </div>
        </div>
        <div class="pages-container" id="pages-container">
            <div class="pages-wrapper" id="pages-wrapper">
                <div class="page" id="left-page-container">
                    <div class="page-content">
                        <img id="left-page" src="" alt="Left Page">
                    </div>
                </div>
                <div class="page" id="right-page-container">
                    <div class="page-content">
                        <img id="right-page" src="" alt="Right Page">
                    </div>
                </div>
            </div>
        </div>
        <div class="page-number left-page-number" id="left-page-number"></div>
        <div class="page-number right-page-number" id="right-page-number"></div>
        <button id="prev-button" class="nav-button" disabled>◀</button>
        <button id="next-button" class="nav-button">▶</button>
        <div class="zoom-controls">
            <button id="zoom-out" class="zoom-btn">-</button>
            <button id="zoom-in" class="zoom-btn">+</button>
            <button id="zoom-reset" class="zoom-btn">↺</button>
        </div>
    </div>
    <script>
        // 图片列表
        const images = """ + js_image_list + """;
        // 子目录列表（用于搜索，排除点文件背景）
        const subdirs = """ + js_subdirs_list + """;
        // 目录标签
        const dirTags = """ + js_dir_tags + """;
        // 全局目录信息 (新增)
        const allDirs = """ + js_all_dirs + """;
        // 返回根目录的路径
        const backToRootPath = '""" + back_to_root_path + """';
        let currentIndex = 0;
        let leftScale = 1;
        let rightScale = 1;
        let isDarkTheme = localStorage.getItem('darkTheme') === 'true';
        let isBackgroundEnabled = localStorage.getItem('backgroundEnabled') !== 'false';
        let isBlurEnabled = localStorage.getItem('blurEnabled') === 'true';
        let isMobileView = window.innerWidth <= 768;
        // 视图控制元素
        const scrollViewBtn = document.getElementById('scroll-view-btn');
        const bookViewBtn = document.getElementById('book-view-btn');
        const scrollView = document.getElementById('scroll-view');
        const bookView = document.getElementById('book-view');
        const themeToggle = document.getElementById('theme-toggle');
        const bgToggle = document.getElementById('bg-toggle');
        const blurToggle = document.getElementById('blur-toggle');
        const body = document.body;
        // 搜索元素 (合并后，移除了 tagSearchBox)
        const searchBox = document.getElementById('search-box');
        const searchResults = document.getElementById('search-results');
        // 双页视图元素
        const exitBookViewBtn = document.getElementById('exit-book-view');
        const prevButton = document.getElementById('prev-button');
        const nextButton = document.getElementById('next-button');
        const pagesContainer = document.getElementById('pages-container');
        const pagesWrapper = document.getElementById('pages-wrapper');
        const leftPage = document.getElementById('left-page');
        const rightPage = document.getElementById('right-page');
        const leftPageContainer = document.getElementById('left-page-container');
        const rightPageContainer = document.getElementById('right-page-container');
        const leftPageNumber = document.getElementById('left-page-number');
        const rightPageNumber = document.getElementById('right-page-number');
        // 缩放控制元素
        const zoomInBtn = document.getElementById('zoom-in');
        const zoomOutBtn = document.getElementById('zoom-out');
        const zoomResetBtn = document.getElementById('zoom-reset');
        // 应用主题
        function applyTheme() {
            if (isDarkTheme) {
                body.classList.add('dark-theme');
                themeToggle.textContent = '☀️ 浅色';
            } else {
                body.classList.remove('dark-theme');
                themeToggle.textContent = '🌙 深色';
            }
            localStorage.setItem('darkTheme', isDarkTheme);
        }
        // 切换主题
        function toggleTheme() {
            isDarkTheme = !isDarkTheme;
            applyTheme();
        }
        // 应用背景
        function applyBackground() {
            if (isBackgroundEnabled) {
                body.classList.add('bg-enabled');
                // 使用相对于根目录的路径
                const img = new Image();
                img.onload = function() {
                    body.style.backgroundImage = `url("${backToRootPath}.dotfile_background.png")`;
                };
                img.onerror = function() {
                    const img2 = new Image();
                    img2.onload = function() {
                        body.style.backgroundImage = `url("${backToRootPath}.dotfile_background.jpg")`;
                    };
                    img2.onerror = function() {
                        body.style.backgroundImage = `url("${backToRootPath}.dotfile_background.webp")`;
                    };
                    img2.src = `${backToRootPath}.dotfile_background.jpg`;
                };
                img.src = `${backToRootPath}.dotfile_background.png`;
                bgToggle.textContent = '🖼️ 隐藏背景';
            } else {
                body.classList.remove('bg-enabled');
                body.style.backgroundImage = 'none';
                bgToggle.textContent = '🖼️ 显示背景';
            }
            localStorage.setItem('backgroundEnabled', isBackgroundEnabled);
            applyBlur(); // 重新应用模糊状态
        }
        // 切换背景
        function toggleBackground() {
            isBackgroundEnabled = !isBackgroundEnabled;
            applyBackground();
        }
        // 应用模糊效果
        function applyBlur() {
            if (isBackgroundEnabled && isBlurEnabled) {
                body.classList.add('bg-blur');
                blurToggle.textContent = '❄️ 关闭模糊';
            } else {
                body.classList.remove('bg-blur');
                blurToggle.textContent = '💧 开启模糊';
            }
            localStorage.setItem('blurEnabled', isBlurEnabled);
        }
        // 切换模糊效果
        function toggleBlur() {
            isBlurEnabled = !isBlurEnabled;
            applyBlur();
        }
        // 检查是否为移动设备
        function checkMobile() {
            isMobileView = window.innerWidth <= 768;
        }
        // 切换视图
        function switchToScrollView() {
            scrollView.style.display = 'block';
            bookView.style.display = 'none';
            scrollViewBtn.classList.add('active');
            bookViewBtn.classList.remove('active');
            document.body.style.overflow = 'auto';
        }
        function switchToBookView() {
            scrollView.style.display = 'none';
            bookView.style.display = 'block';
            bookViewBtn.classList.add('active');
            scrollViewBtn.classList.remove('active');
            document.body.style.overflow = 'hidden';
            checkMobile();
            resetView();
            showPages();
        }
        // 退出双页视图
        function exitBookView() {
            switchToScrollView();
        }
        // 重置视图（缩放）
        function resetView() {
            leftScale = 1;
            rightScale = 1;
            applyTransform();
        }
        // 应用变换（独立缩放）
        function applyTransform() {
            leftPage.style.transform = `scale(${leftScale})`;
            rightPage.style.transform = `scale(${rightScale})`;
        }
        // 显示双页内容
        function showPages() {
            checkMobile();
            if (currentIndex < images.length) {
                leftPage.src = images[currentIndex];
                leftPageNumber.textContent = `第 ${currentIndex + 1} 页`;
                leftPage.style.display = 'block';
                leftPageNumber.style.display = 'block';
                // 移动端显示单页（全屏）
                if (isMobileView) {
                    leftPageContainer.style.maxWidth = '95%';
                    rightPageContainer.style.display = 'none';
                } else {
                    leftPageContainer.style.maxWidth = '45%';
                    rightPageContainer.style.display = 'flex';
                }
            } else {
                leftPage.style.display = 'none';
                leftPageNumber.style.display = 'none';
            }
            if (!isMobileView && currentIndex + 1 < images.length) {
                rightPage.src = images[currentIndex + 1];
                rightPageNumber.textContent = `第 ${currentIndex + 2} 页`;
                rightPage.style.display = 'block';
                rightPageNumber.style.display = 'block';
            } else {
                rightPage.style.display = 'none';
                rightPageNumber.style.display = 'none';
            }
            // 更新按钮状态
            prevButton.disabled = currentIndex === 0;
            nextButton.disabled = currentIndex >= images.length - (isMobileView ? 1 : 2);
            // 重置视图
            resetView();
        }
        // 导航功能
        function goToPrev() {
            checkMobile();
            currentIndex = Math.max(0, currentIndex - (isMobileView ? 1 : 2));
            showPages();
        }
        function goToNext() {
            checkMobile();
            currentIndex = Math.min(images.length - 1, currentIndex + (isMobileView ? 1 : 2));
            showPages();
        }
        // 缩放功能（独立缩放）
        function zoomIn() {
            if (!isMobileView) {
                // 双页模式：同时放大两张图片
                leftScale = Math.min(5, leftScale + 0.1);
                rightScale = Math.min(5, rightScale + 0.1);
            } else {
                // 单页模式：只放大当前图片
                leftScale = Math.min(5, leftScale + 0.1);
            }
            applyTransform();
        }
        function zoomOut() {
            if (!isMobileView) {
                // 双页模式：同时缩小两张图片
                leftScale = Math.max(0.3, leftScale - 0.1);
                rightScale = Math.max(0.3, rightScale - 0.1);
            } else {
                // 单页模式：只缩小当前图片
                leftScale = Math.max(0.3, leftScale - 0.1);
            }
            applyTransform();
        }
        // --- 新增/修改的全局搜索功能 ---
        // 辅助函数：计算从当前页面到目标目录的相对路径
        function getRelativePathToTarget(targetPathStr) {
             // 当前页面相对于根的路径 (不包含文件名)
            const currentPathStr = backToRootPath ? backToRootPath.replace(/\\/$/, '') : '.';
            
            // 如果目标是根目录
            if (targetPathStr === '.') {
                 // 计算需要向上返回的层级数
                const upLevels = (currentPathStr.match(/\\.\\./g) || []).length;
                return '../'.repeat(upLevels) || './';
            }
            
            // 如果当前就在根目录
            if (currentPathStr === '.') {
                return targetPathStr + '/';
            }
            
            // 计算共同前缀长度
            const currentParts = currentPathStr.split('/');
            const targetParts = targetPathStr.split('/');
            
            let commonLength = 0;
            const minLength = Math.min(currentParts.length, targetParts.length);
            while (commonLength < minLength && currentParts[commonLength] === targetParts[commonLength]) {
                commonLength++;
            }
            
            // 计算需要向上返回的步数
            const upSteps = currentParts.length - commonLength;
            // 计算需要向下进入的路径
            const downPath = targetParts.slice(commonLength).join('/');
            
            // 构建相对路径
            let relativePath = '../'.repeat(upSteps);
            if (downPath) {
                relativePath += downPath + '/';
            } else if (relativePath === '') {
                 // 如果路径相同
                relativePath = './';
            }
            
            return relativePath;
        }
        
        // 全局搜索功能 (合并目录名和标签搜索，并优化结果)
        function performGlobalSearch() {
            const query = searchBox.value.toLowerCase().trim();
            // 清空之前的搜索结果
            searchResults.innerHTML = '';
            // 如果查询为空，隐藏搜索结果
            if (query === '') {
                searchResults.style.display = 'none';
                return;
            }
            
            // 过滤匹配的全局子目录
            const matches = [];
            for (const [path, dirInfo] of Object.entries(allDirs)) {
                let isMatch = false;
                let matchType = 'directory'; // 'directory' or 'tag'
                
                // 1. 搜索目录显示名称
                if (dirInfo.clean_name.toLowerCase().includes(query)) {
                    isMatch = true;
                    matchType = 'directory';
                } 
                // 2. 搜索标签
                else if (dirInfo.tags && dirInfo.tags.some(tag => tag.toLowerCase().includes(query))) {
                    isMatch = true;
                    matchType = 'tag';
                }
                
                if (isMatch) {
                    matches.push({path, matchType, ...dirInfo});
                }
            }
            
            // --- 优化搜索结果：过滤掉有父目录在结果中的项 ---
            // 1. 先按路径深度排序，确保父目录在前
            matches.sort((a, b) => {
                const depthA = a.path === '.' ? 0 : a.path.split('/').length;
                const depthB = b.path === '.' ? 0 : b.path.split('/').length;
                return depthA - depthB;
            });
            
            // 2. 标记哪些项的父目录在结果中
            const matchPaths = new Set(matches.map(m => m.path));
            for (let i = 0; i < matches.length; i++) {
                const currentPath = matches[i].path;
                if (currentPath === '.') continue; // 根目录没有父目录
                
                const parentPath = currentPath.split('/').slice(0, -1).join('/') || '.';
                if (matchPaths.has(parentPath)) {
                    matches[i].has_parent_in_results = true;
                }
            }
            
            // 3. 过滤掉有父目录在结果中的项 (保留最顶层)
            const filteredMatches = matches.filter(match => !match.has_parent_in_results);
            
            // --- 显示匹配结果 ---
            if (filteredMatches.length > 0) {
                filteredMatches.forEach(dir => {
                    const resultItem = document.createElement('div');
                    resultItem.className = 'search-result-item';
                    
                    // 计算正确的相对链接
                    const href = getRelativePathToTarget(dir.path);
                    
                    // 根据匹配类型决定显示内容
                    let displayText = dir.clean_name;
                    if (dir.matchType === 'tag') {
                        // 如果是标签匹配，添加标签图标
                        displayText = `<span class="search-result-tag-match-icon">🏷️</span>${displayText}`;
                    }
                    
                    resultItem.innerHTML = `
                        <a href="${href}">${displayText}</a>
                        <div class="search-result-path">${dir.full_path}</div>
                    `;
                    // 添加标签显示 (如果有的话)
                    if (dir.tags && dir.tags.length > 0) {
                        const tagsContainer = document.createElement('div');
                        tagsContainer.className = 'search-result-tags';
                        dir.tags.forEach(tag => {
                            const tagElement = document.createElement('span');
                            tagElement.className = 'search-result-tag';
                            tagElement.textContent = tag;
                            tagsContainer.appendChild(tagElement);
                        });
                        resultItem.appendChild(tagsContainer);
                    }
                    // 为整个结果项添加点击事件监听器
                    resultItem.addEventListener('click', (e) => {
                         // 阻止默认的 <a> 标签跳转，因为我们自己处理
                        e.preventDefault(); 
                         // 使用 window.location 进行跳转
                        window.location.href = href; 
                    });
                    searchResults.appendChild(resultItem);
                });
                searchResults.style.display = 'block';
            } else {
                // 没有匹配结果
                const noResultItem = document.createElement('div');
                noResultItem.className = 'search-result-item';
                noResultItem.textContent = '未找到匹配的目录或标签';
                searchResults.appendChild(noResultItem);
                searchResults.style.display = 'block';
            }
        }
        // --- 修改结束 ---
        // 事件监听器
        scrollViewBtn.addEventListener('click', switchToScrollView);
        bookViewBtn.addEventListener('click', switchToBookView);
        exitBookViewBtn.addEventListener('click', exitBookView);
        prevButton.addEventListener('click', goToPrev);
        nextButton.addEventListener('click', goToNext);
        zoomInBtn.addEventListener('click', zoomIn);
        zoomOutBtn.addEventListener('click', zoomOut);
        zoomResetBtn.addEventListener('click', resetView);
        themeToggle.addEventListener('click', toggleTheme);
        bgToggle.addEventListener('click', toggleBackground);
        blurToggle.addEventListener('click', toggleBlur);
        // 搜索框事件 (合并后，只监听一个框)
        searchBox.addEventListener('input', performGlobalSearch);
        // 点击其他地方隐藏搜索结果 (更新了监听器列表)
        document.addEventListener('click', (e) => {
            if (!searchBox.contains(e.target) && 
                !searchResults.contains(e.target)) {
                searchResults.style.display = 'none';
            }
        });
        // 响应窗口大小变化
        window.addEventListener('resize', function() {
            if (bookView.style.display !== 'none') {
                showPages();
            }
        });
        // 键盘导航
        document.addEventListener('keydown', function(e) {
            if (bookView.style.display !== 'none') {
                if (e.key === 'Escape') {
                    exitBookView();
                } else if (e.key === 'ArrowLeft') {
                    goToPrev();
                } else if (e.key === 'ArrowRight') {
                    goToNext();
                } else if (e.key === '+' || e.key === '=') {
                    zoomIn();
                } else if (e.key === '-') {
                    zoomOut();
                } else if (e.key === '0') {
                    resetView();
                }
            }
        });
        // 鼠标滚轮缩放
        pagesContainer.addEventListener('wheel', function(e) {
            if (bookView.style.display !== 'none') {
                e.preventDefault();
                if (e.deltaY < 0) {
                    zoomIn();
                } else {
                    zoomOut();
                }
            }
        });
        // 双指缩放支持（移动端）
        let initialDistance = 0;
        let initialScale = 1;
        let currentScaleTarget = null;
        pagesContainer.addEventListener('touchstart', function(e) {
            if (e.touches.length === 2) {
                initialDistance = Math.hypot(
                    e.touches[0].clientX - e.touches[1].clientX,
                    e.touches[0].clientY - e.touches[1].clientY
                );
                // 确定缩放目标
                if (e.touches[0].clientX < window.innerWidth / 2 || e.touches[1].clientX < window.innerWidth / 2) {
                    currentScaleTarget = 'left';
                    initialScale = leftScale;
                } else {
                    currentScaleTarget = 'right';
                    initialScale = rightScale;
                }
            }
        });
        pagesContainer.addEventListener('touchmove', function(e) {
            if (e.touches.length === 2 && currentScaleTarget) {
                e.preventDefault();
                const currentDistance = Math.hypot(
                    e.touches[0].clientX - e.touches[1].clientX,
                    e.touches[0].clientY - e.touches[1].clientY
                );
                if (initialDistance > 0) {
                    const scale = initialScale * (currentDistance / initialDistance);
                    const clampedScale = Math.min(5, Math.max(0.3, scale));
                    if (currentScaleTarget === 'left') {
                        leftScale = clampedScale;
                    } else if (currentScaleTarget === 'right') {
                        rightScale = clampedScale;
                    }
                    applyTransform();
                }
            }
        });
        pagesContainer.addEventListener('touchend', function() {
            initialDistance = 0;
            currentScaleTarget = null;
        });
        // 初始化
        applyTheme();
        applyBackground();
        applyBlur();
    </script>
</body>
</html>"""
    # 写入 index.html 文件
    index_file_path = directory_path / 'index.html'
    try:
        with open(index_file_path, 'w', encoding='utf-8') as f:
            f.write(html_content)
        print(f"已生成: {index_file_path}")
    except Exception as e:
        print(f"错误: 无法写入文件 '{index_file_path}': {e}")
        return
    # 递归处理子目录 (包括隐藏目录)
    for subdir_name in subdirectories:
        subdir_path = directory_path / subdir_name
        generate_index_html(subdir_path)

if __name__ == "__main__":
    current_dir = Path.cwd()
    print(f"开始为目录 '{current_dir}' 及其子目录生成图片库...")
    generate_index_html(current_dir)
    print("完成！在浏览器中打开 index.html 查看结果。")
