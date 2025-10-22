# Awesome_Script

* Novelcrawler: 需要chromedriver,以及类似`<div id="content">`这样的标签(指带"content"的)才可以爬取

* Creatmangapages: 漫画网页阅读器,遍历当前的目录,在每个目录生成`index.html`,如果有图片显示图片(`.dotfile_background`这个隐藏文件名用来作为背景图片的),注意别在类似`~/`运行-除非你想整个`$HOME`每个文件夹都有一个`index.html`,注:无法在本地正常运行

* TxttoEpub: txt转epub ,使用方法:`python TxttoEpub.py 需要转换的文件.txt(文本编码尽量为标准Utf-8[建议]/标准GBK) 输出文件.epub epub书封图片` (注:会自动分割章节,如果分割失败在rules.txt复制几个到`CHAPTER_PATTERNS`)

* 依赖: 忘了...喂给AI问问?