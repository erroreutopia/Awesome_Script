# Awesome_Script

* Novelcrawler: 需要chromedriver,以及类似`<div id="content">`这样的标签(指带"content"的)才可以爬取

* Creatmangapages: 漫画网页阅读器,遍历当前的目录,在每个目录生成`index.html`,如果有图片显示图片(`.dotfile_background`这个隐藏文件名用来作为背景图片的),注意别在类似`~/`运行-除非你想整个`$HOME`每个文件夹都有一个`index.html`,注:无法在本地正常运行

* TxttoEpub: txt转epub ,使用方法:`python TxttoEpub.py 需要转换的文件.txt(文本编码尽量为标准Utf-8[建议]/标准GBK) 输出文件.epub epub书封图片` (注:会自动分割章节,如果分割失败在rules.txt复制几个到`CHAPTER_PATTERNS`)

* agamepack：打包PRGMaker,Wine为Appimage（注：RPG存档位置为：$HOME/Game/HTMLGame/NWJS/$(GAMENAME)/，WINE存档位置为$HOME/Game/WineGame/Save/$(GAMENAME)），wine如果需要在游戏目录而非.wine目录会无法保存！以及，Nwjs和proton-ge前者需要在$HOME/App/nwjs-sdk/,后者需要在`PATH`
使用方法：agamepack -r 目录，没有默认为game目录。-i icon文件。 -n 游戏名称。-o 输出的文件名，只能是ACSII字符 --wine-exec 设置wine的可执行文件名，但是脚本会自动识别，可无视。--wine-cmd 设置wine/proton等兼容层路径，默认为proton-ge, 
更多查看 `-h`选项
* 依赖: 忘了...喂给AI问问?