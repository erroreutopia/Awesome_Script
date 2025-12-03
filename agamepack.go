package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 配置结构体
type Config struct {
	GameSourceDir    string
	AppName          string
	IconPath         string
	PackageType      string
	WineExec         string
	WineCmd          string
	WineSaveDir      string
	SavePattern      string
	SaveStart        int
	SaveEnd          int
	AutoBuild        bool
	ForceBuild       bool
	OutputFilename   string
	NWJSPath         string
	SaveBaseDir      string
	WineArchiveBaseDir string
}

var cfg Config

func main() {
	// 初始化默认配置
	cfg = Config{
		WineCmd:          "proton-ge",
		SavePattern:      "Save%d",
		SaveStart:        1,
		SaveEnd:          10,
		NWJSPath:         filepath.Join(os.Getenv("HOME"), "App/nwjs-sdk/nw"),
		SaveBaseDir:      filepath.Join(os.Getenv("HOME"), "Game/HTMLGame/NWJS/SAVE"),
		WineArchiveBaseDir: filepath.Join(os.Getenv("HOME"), "Game/WineGame/Save"),
	}

	// 设置命令行标志
	gameDir := flag.String("r", "", "游戏源目录")
	gameDirLong := flag.String("game-dir", "", "游戏源目录")
	name := flag.String("n", "", "应用名称")
	nameLong := flag.String("name", "", "应用名称")
	icon := flag.String("i", "", "自定义图标文件")
	iconLong := flag.String("icon", "", "自定义图标文件")
	pkgType := flag.String("t", "", "包类型 (nwjs/wine)")
	pkgTypeLong := flag.String("type", "", "包类型 (nwjs/wine)")
	wineExec := flag.String("wine-exec", "", "Wine可执行文件")
	wineCmd := flag.String("wine-cmd", "proton-ge", "Wine命令")
	wineSaveDir := flag.String("wine-save", "", "Wine存档目录")
	output := flag.String("o", "", "输出文件名")
	outputLong := flag.String("output", "", "输出文件名")
	savePattern := flag.String("save-pattern", "Save%d", "自定义存档模式")
	saveStart := flag.Int("save-start", 1, "起始编号")
	saveEnd := flag.Int("save-end", 10, "结束编号")
	autoBuild := flag.Bool("b", false, "自动构建，不询问")
	autoBuildLong := flag.Bool("build", false, "自动构建，不询问")
	forceBuild := flag.Bool("y", false, "跳过所有确认，强制执行")
	forceBuildLong := flag.Bool("yes", false, "跳过所有确认，强制执行")
	help := flag.Bool("h", false, "显示帮助信息")
	helpLong := flag.Bool("help", false, "显示帮助信息")

	// 解析命令行参数
	flag.Parse()

	// 处理帮助请求
	if *help || *helpLong {
		showHelp()
		return
	}

	// 设置配置
	cfg.GameSourceDir = *gameDir
	if cfg.GameSourceDir == "" {
		cfg.GameSourceDir = *gameDirLong
	}

	cfg.AppName = *name
	if cfg.AppName == "" {
		cfg.AppName = *nameLong
	}

	cfg.IconPath = *icon
	if cfg.IconPath == "" {
		cfg.IconPath = *iconLong
	}

	cfg.PackageType = *pkgType
	if cfg.PackageType == "" {
		cfg.PackageType = *pkgTypeLong
	}

	cfg.WineExec = *wineExec
	cfg.WineCmd = *wineCmd
	cfg.WineSaveDir = *wineSaveDir
	cfg.OutputFilename = *output
	if cfg.OutputFilename == "" {
		cfg.OutputFilename = *outputLong
	}

	cfg.SavePattern = *savePattern
	cfg.SaveStart = *saveStart
	cfg.SaveEnd = *saveEnd
	cfg.AutoBuild = *autoBuild
	if !cfg.AutoBuild {
		cfg.AutoBuild = *autoBuildLong
	}
	cfg.ForceBuild = *forceBuild
	if !cfg.ForceBuild {
		cfg.ForceBuild = *forceBuildLong
	}

	// 检查是否需要问卷式模式
	if shouldUseInteractiveMode() {
		runInteractiveMode()
	} else {
		// 有参数模式，补全缺失的必要参数
		completeConfig()
	}

	// 验证配置
	if err := validateConfig(); err != nil {
		fmt.Printf("❌ 配置错误: %v\n", err)
		os.Exit(1)
	}

	// 构建AppImage
	buildAppImage()
}

func showHelp() {
	fmt.Println("用法: agamepack [选项]")
	fmt.Println("")
	fmt.Println("构建AppImage游戏包，支持NW.js和Wine/Windows游戏")
	fmt.Println("支持目录重定向和自定义存档模式，100%只读文件系统安全")
	fmt.Println("")
	fmt.Println("选项:")
	fmt.Println("  -r, --game-dir DIR     游戏源目录")
	fmt.Println("  -n, --name NAME        应用名称")
	fmt.Println("  -i, --icon FILE        自定义图标文件")
	fmt.Println("  -t, --type TYPE        包类型 (nwjs/wine)")
	fmt.Println("  -o, --output FILE      输出文件名")
	fmt.Println("  -b, --build            自动构建，不询问")
	fmt.Println("  -y, --yes              跳过所有确认，强制执行")
	fmt.Println("  -h, --help             显示此帮助信息")
	fmt.Println("")
	fmt.Println("Wine专用选项:")
	fmt.Println("  --wine-exec FILE       Wine可执行文件")
	fmt.Println("  --wine-cmd CMD         Wine命令 (默认: proton-ge)")
	fmt.Println("  --wine-save DIR        Wine存档目录")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  # 问卷式模式 (无参数)")
	fmt.Println("  agamepack")
	fmt.Println("")
	fmt.Println("  # 指定参数")
	fmt.Println("  agamepack -r \"觅长生\" -n \"觅长生\" -i \"觅长生/icon.png\" \\")
	fmt.Println("    --wine-exec \"觅长生.exe\" --wine-save MCSSave --build -y")
}

func shouldUseInteractiveMode() bool {
	// 无参数且未设置必要配置时使用问卷式
	return len(os.Args) == 1 && cfg.GameSourceDir == "" && cfg.AppName == ""
}

func runInteractiveMode() {
	fmt.Println("📋 进入问卷式设置...")
	fmt.Println("")

	// 1. 游戏目录
	for {
		var dir string
		fmt.Print("游戏源目录 (例如: ./game 或 /path/to/game): ")
		fmt.Scanln(&dir)
		if dir == "" {
			dir = "./"
		}

		absPath, err := filepath.Abs(dir)
		if err != nil {
			fmt.Printf("❌ 路径错误: %v\n", err)
			continue
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Printf("❌ 目录不存在: %s\n", absPath)
			var create string
			fmt.Print("创建此目录? [y/N]: ")
			fmt.Scanln(&create)
			if strings.ToLower(create) == "y" {
				if err := os.MkdirAll(absPath, 0755); err != nil {
					fmt.Printf("❌ 创建目录失败: %v\n", err)
					continue
				}
				fmt.Printf("✅ 目录已创建: %s\n", absPath)
				cfg.GameSourceDir = absPath
				break
			}
		} else {
			fmt.Printf("✅ 目录存在: %s\n", absPath)
			cfg.GameSourceDir = absPath
			break
		}
	}

	// 2. 应用名称
	defaultName := filepath.Base(cfg.GameSourceDir)
	fmt.Printf("应用名称 (默认: %s): ", defaultName)
	var name string
	fmt.Scanln(&name)
	if name == "" {
		name = defaultName
	}
	cfg.AppName = name

	// 3. 游戏类型
	fmt.Println("")
	fmt.Println("游戏类型:")
	fmt.Println("1. NW.js/HTML5 游戏 (package.json 或 index.html)")
	fmt.Println("2. Wine/Windows 游戏 (*.exe 文件)")
	fmt.Println("3. RPG Maker 游戏 (www/ 目录)")

	for {
		var choice string
		fmt.Print("选择类型 [1-3]: ")
		fmt.Scanln(&choice)
		if choice == "" {
			choice = "1"
		}

		switch choice {
		case "1":
			cfg.PackageType = "nwjs"
			fmt.Println("✅ 选择: NW.js/HTML5 游戏")
			return
		case "2", "3":
			cfg.PackageType = "wine"
			fmt.Println("✅ 选择: Wine/Windows 游戏")
			setupWineInteractive()
			return
		default:
			fmt.Println("❌ 无效选择，请输入 1-3")
		}
	}
}

func setupWineInteractive() {
	// 4. 可执行文件
	fmt.Println("")
	fmt.Println("🔍 检测可执行文件...")
	exeFiles := findExeFiles(cfg.GameSourceDir)
	if len(exeFiles) > 0 {
		fmt.Println("检测到可执行文件:")
		for i, file := range exeFiles {
			fmt.Printf("  %d. %s\n", i+1, file)
		}

		for {
			var choice string
			fmt.Printf("选择可执行文件 [1]: ")
			fmt.Scanln(&choice)
			if choice == "" {
				choice = "1"
			}

			index := 1
			if choice != "" {
				parsed, err := strconv.Atoi(choice)
				if err == nil {
					index = parsed
				}
			}

			if index >= 1 && index <= len(exeFiles) {
				cfg.WineExec = exeFiles[index-1]
				fmt.Printf("✅ 选择: %s\n", cfg.WineExec)
				break
			} else {
				fmt.Printf("❌ 无效选择，请输入 1-%d\n", len(exeFiles))
			}
		}
	} else {
		for {
			var exec string
			fmt.Print("Wine可执行文件 (例如: game.exe): ")
			fmt.Scanln(&exec)
			if exec != "" {
				cfg.WineExec = exec
				break
			}
			fmt.Println("❌ 请输入可执行文件名")
		}
	}

	// 5. 存档设置
	fmt.Println("")
	fmt.Println("存档设置:")
	fmt.Println("1. 目录重定向 (推荐: save/, MCSSave/ 等)")
	fmt.Println("2. 自定义文件模式 (Save01, Save02...)")

	for {
		var choice string
		fmt.Print("选择存档方式 [1]: ")
		fmt.Scanln(&choice)
		if choice == "" {
			choice = "1"
		}

		switch choice {
		case "1":
			setupWineSaveDirInteractive()
			return
		case "2":
			setupCustomSavePatternInteractive()
			return
		default:
			fmt.Println("❌ 无效选择，请输入 1-2")
		}
	}
}

func setupWineSaveDirInteractive() {
	fmt.Println("🔍 检测存档目录...")
	saveDirs := findSaveDirectories(cfg.GameSourceDir)
	if len(saveDirs) > 0 {
		fmt.Println("检测到可能的存档目录:")
		for i, dir := range saveDirs {
			fmt.Printf("  %d. %s\n", i+1, dir)
		}

		for {
			var choice string
			fmt.Printf("选择存档目录 [1]: ")
			fmt.Scanln(&choice)
			if choice == "" {
				choice = "1"
			}

			index := 1
			if choice != "" {
				parsed, err := strconv.Atoi(choice)
				if err == nil {
					index = parsed
				}
			}

			if index >= 1 && index <= len(saveDirs) {
				cfg.WineSaveDir = saveDirs[index-1]
				fmt.Printf("✅ 选择存档目录: %s\n", cfg.WineSaveDir)
				return
			} else {
				fmt.Printf("❌ 无效选择，请输入 1-%d\n", len(saveDirs))
			}
		}
	} else {
		for {
			var dir string
			fmt.Print("存档目录 (例如: save): ")
			fmt.Scanln(&dir)
			if dir != "" {
				cfg.WineSaveDir = dir
				return
			}
			fmt.Println("❌ 请输入存档目录名")
		}
	}
}

func setupCustomSavePatternInteractive() {
	fmt.Print("存档文件模式 (默认: Save%02d.rvdata2): ")
	var pattern string
	fmt.Scanln(&pattern)
	if pattern == "" {
		pattern = "Save%02d.rvdata2"
	}
	cfg.SavePattern = pattern

	fmt.Print("起始编号 (默认: 1): ")
	var startStr string
	fmt.Scanln(&startStr)
	start := 1
	if startStr != "" {
		parsed, _ := strconv.Atoi(startStr)
		if parsed > 0 {
			start = parsed
		}
	}
	cfg.SaveStart = start

	fmt.Print("结束编号 (默认: 9): ")
	var endStr string
	fmt.Scanln(&endStr)
	end := 9
	if endStr != "" {
		parsed, _ := strconv.Atoi(endStr)
		if parsed > 0 {
			end = parsed
		}
	}
	cfg.SaveEnd = end
}

func completeConfig() {
	// 补全游戏目录
	if cfg.GameSourceDir == "" {
		fmt.Println("❓ 未指定游戏目录，尝试检测...")
		possibleDirs := []string{"./", "game", "dist", "build", "www"}
		for _, dir := range possibleDirs {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				absPath, _ := filepath.Abs(dir)
				cfg.GameSourceDir = absPath
				fmt.Printf("✅ 检测到目录: %s\n", cfg.GameSourceDir)
				break
			}
		}
		if cfg.GameSourceDir == "" {
			fmt.Println("❌ 无法找到有效游戏目录")
			os.Exit(1)
		}
	} else {
		absPath, err := filepath.Abs(cfg.GameSourceDir)
		if err != nil {
			fmt.Printf("❌ 路径错误: %v\n", err)
			os.Exit(1)
		}
		cfg.GameSourceDir = absPath
	}

	// 补全应用名称
	if cfg.AppName == "" {
		cfg.AppName = filepath.Base(cfg.GameSourceDir)
		fmt.Printf("✅ 使用目录名作为应用名称: %s\n", cfg.AppName)
	}

	// 补全包类型
	if cfg.PackageType == "" {
		if isNWJSApp(cfg.GameSourceDir) {
			cfg.PackageType = "nwjs"
			fmt.Println("✅ 自动检测: NW.js 应用")
		} else if isWineApp(cfg.GameSourceDir) {
			cfg.PackageType = "wine"
			fmt.Println("✅ 自动检测: Wine/Windows 应用")
		} else {
			cfg.PackageType = "nwjs"
			fmt.Println("⚠️  无法确定类型，使用默认: NW.js")
		}
	}

	// 补全Wine可执行文件
	if cfg.PackageType == "wine" && cfg.WineExec == "" {
		exeFiles := findExeFiles(cfg.GameSourceDir)
		if len(exeFiles) > 0 {
			cfg.WineExec = exeFiles[0]
			fmt.Printf("✅ 检测到可执行文件: %s\n", cfg.WineExec)
		} else {
			cfg.WineExec = "game.exe"
			fmt.Printf("⚠️  未指定可执行文件，使用默认: %s\n", cfg.WineExec)
		}
	}

	// 补全输出文件名
	if cfg.OutputFilename == "" {
		cleanName := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
				return r
			}
			return -1
		}, cfg.AppName)
		if cleanName == "" {
			cleanName = "Game"
		}
		cfg.OutputFilename = cleanName + ".AppImage"
		fmt.Printf("📝 使用目录名作为默认文件名: %s\n", cfg.OutputFilename)
	} else if !strings.HasSuffix(cfg.OutputFilename, ".AppImage") {
		cfg.OutputFilename += ".AppImage"
	}
}

func validateConfig() error {
	// 验证游戏目录
	if _, err := os.Stat(cfg.GameSourceDir); os.IsNotExist(err) {
		return fmt.Errorf("游戏目录不存在: %s", cfg.GameSourceDir)
	}

	// 验证包类型
	switch cfg.PackageType {
	case "nwjs", "wine":
		// 有效类型
	default:
		return fmt.Errorf("不支持的包类型: %s", cfg.PackageType)
	}

	// 验证Wine配置
	if cfg.PackageType == "wine" {
		if cfg.WineExec == "" {
			return fmt.Errorf("Wine可执行文件未指定")
		}
	}

	return nil
}

func buildAppImage() {
	fmt.Printf("📂 复制游戏文件: %s -> build/%s.AppDir/game\n", cfg.GameSourceDir, cfg.AppName)

	// 创建目录结构
	appDir := filepath.Join("build", cfg.AppName+".AppDir")
	gameSubDir := filepath.Join(appDir, "game")
	os.RemoveAll("build")
	os.MkdirAll(gameSubDir, 0755)

	// 复制游戏文件
	copyDir(cfg.GameSourceDir, gameSubDir)

	// 存档处理
	if cfg.PackageType == "wine" {
		fmt.Println("🎯 Wine应用: 存档处理")
		wineArchiveDir := filepath.Join(cfg.WineArchiveBaseDir, cfg.AppName)
		os.MkdirAll(wineArchiveDir, 0755)
		fmt.Printf("📁 固定Archive目录: %s\n", wineArchiveDir)

		if cfg.WineSaveDir != "" {
			// 目录重定向模式
			fmt.Printf("🔗 目录重定向模式: %s/\n", cfg.WineSaveDir)
			wineSavePath := filepath.Join(gameSubDir, cfg.WineSaveDir)
			targetSavePath := filepath.Join(wineArchiveDir, cfg.WineSaveDir)
			os.MkdirAll(filepath.Dir(targetSavePath), 0755)

			// 创建符号链接
			os.Remove(wineSavePath)
			os.Symlink(targetSavePath, wineSavePath)
			fmt.Printf("🔗 %s -> %s\n", wineSavePath, targetSavePath)
			fmt.Println("✅ 目录重定向完成")
		} else {
			// 自定义文件模式
			fmt.Printf("🔗 创建自定义存档链接: %s (%d to %d)\n",
				cfg.SavePattern, cfg.SaveStart, cfg.SaveEnd)

			totalLinks := 0
			for i := cfg.SaveStart; i <= cfg.SaveEnd; i++ {
				filename := fmt.Sprintf(cfg.SavePattern, i)
				sourceFile := filepath.Join(gameSubDir, filename)
				targetFile := filepath.Join(wineArchiveDir, filename)

				if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
					os.MkdirAll(filepath.Dir(targetFile), 0755)
					os.Remove(sourceFile)
					os.Symlink(targetFile, sourceFile)
					fmt.Printf("🔗 %s -> %s\n", sourceFile, targetFile)
					totalLinks++
				}
			}
			fmt.Printf("✅ 总共预创建 %d 个存档链接\n", totalLinks)
		}
	} else {
		// NW.js: 统一存档目录
		gameSaveDir := filepath.Join(cfg.SaveBaseDir, cfg.AppName)
		os.MkdirAll(gameSaveDir, 0755)

		// 创建标准链接
		createLink(gameSaveDir, filepath.Join(appDir, "save"))
		createLink(gameSaveDir, filepath.Join(gameSubDir, "save"))
		os.MkdirAll(filepath.Join(gameSubDir, "www"), 0755)
		createLink(gameSaveDir, filepath.Join(gameSubDir, "www", "save"))
	}

	// 创建AppRun
	createAppRun(appDir)

	// 创建.desktop
	createDesktopFile(appDir)

	// 创建图标
	createIconFile(appDir)

	// 构建AppImage
	if cfg.AutoBuild || cfg.ForceBuild || askForConfirmation("构建AppImage? [Y/n]: ", true) {
		buildWithAppImageTool(appDir)
	}
}

func isNWJSApp(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
		return true
	}
	return false
}

func isWineApp(dir string) bool {
	var exeFiles []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".exe") {
			exeFiles = append(exeFiles, d.Name())
		}
		return nil
	})
	return len(exeFiles) > 0
}

func findExeFiles(dir string) []string {
	var exeFiles []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".exe") {
			relPath, _ := filepath.Rel(dir, path)
			exeFiles = append(exeFiles, relPath)
		}
		return nil
	})
	return exeFiles
}

func findSaveDirectories(dir string) []string {
	var saveDirs []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			dirName := strings.ToLower(filepath.Base(path))
			if dirName == "save" || dirName == "saves" || dirName == "data" || dirName == "userdata" {
				relPath, _ := filepath.Rel(dir, path)
				saveDirs = append(saveDirs, relPath)
			}
		}
		return nil
	})
	return saveDirs
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func createLink(target string, link string) {
	os.Remove(link)
	os.MkdirAll(filepath.Dir(link), 0755)
	os.Symlink(target, link)
	fmt.Printf("🔗 %s -> %s\n", link, target)
}

func createAppRun(appDir string) {
	appRunPath := filepath.Join(appDir, "AppRun")
	var content string

	if cfg.PackageType == "wine" {
		content = fmt.Sprintf(`#!/bin/bash
# AppRun - Wine专用
APPDIR="$(dirname "$(readlink -f "$0")")"
DESKTOP_FILE=$(find "${APPDIR}" -name "*.desktop" -print -quit 2>/dev/null)
APP_NAME="%s"
[ -n "${DESKTOP_FILE}" ] && APP_NAME=$(grep -i "^Name=" "${DESKTOP_FILE}" | head -1 | cut -d'=' -f2)

# 固定Archive目录
WINE_ARCHIVE_DIR="%s/${APP_NAME}"
mkdir -p "${WINE_ARCHIVE_DIR}" 2>/dev/null || true

# 运行游戏
cd "${APPDIR}/game"
WINE_CMD="%s"
[ ! -x "$(command -v ${WINE_CMD})" ] && WINE_CMD="wine"
exec "${WINE_CMD}" "%s"
`, cfg.AppName, cfg.WineArchiveBaseDir, cfg.WineCmd, cfg.WineExec)
	} else {
		content = fmt.Sprintf(`#!/bin/bash
APPDIR="$(dirname "$(readlink -f "$0")")"
DESKTOP_FILE=$(find "${APPDIR}" -name "*.desktop" -print -quit 2>/dev/null)
APP_NAME="%s"
[ -n "${DESKTOP_FILE}" ] && APP_NAME=$(grep -i "^Name=" "${DESKTOP_FILE}" | head -1 | cut -d'=' -f2)
SAVE_DIR="%s/${APP_NAME}"
mkdir -p "${SAVE_DIR}"
NWJS_PATH="%s"
[ ! -x "${NWJS_PATH}" ] && NWJS_PATH=$(command -v nw 2>/dev/null || echo "nw")
cd "${APPDIR}/game"
exec "${NWJS_PATH}" . --no-sandbox
`, cfg.AppName, cfg.SaveBaseDir, cfg.NWJSPath)
	}

	os.WriteFile(appRunPath, []byte(content), 0755)
}

func createDesktopFile(appDir string) {
	desktopPath := filepath.Join(appDir, cfg.AppName+".desktop")
	content := fmt.Sprintf(`[Desktop Entry]
Name=%s
Exec=AppRun
Icon=%s
Terminal=false
Type=Application
Categories=Game;
X-AppImage-Version=1.0
X-AppImage-Type=%s
`, cfg.AppName, cfg.AppName, cfg.PackageType)
	os.WriteFile(desktopPath, []byte(content), 0644)
}

func createIconFile(appDir string) {
	iconPath := filepath.Join(appDir, cfg.AppName+".png")

	if cfg.IconPath != "" {
		if file, err := os.Open(cfg.IconPath); err == nil {
			defer file.Close()
			targetFile, err := os.Create(iconPath)
			if err == nil {
				defer targetFile.Close()
				io.Copy(targetFile, file)
				fmt.Printf("🖼️  使用自定义图标: %s\n", cfg.IconPath)
				return
			}
		}
	}

	fmt.Println("🎨 生成默认图标...")
	symbol := getFirstTwoLetters(cfg.AppName)
	
	// 修复类型错误: 将UnixNano()和Unix()结果都转换为int
	nanoPart := int(time.Now().UnixNano()) % 0xFFFFFF
	unixPart := int(time.Now().Unix()) % 0xFFFFFF
	colorHex := fmt.Sprintf("%06x", (nanoPart + unixPart) % 0xFFFFFF)
	color := "#" + colorHex

	// 尝试调用convert命令
	if _, err := os.Stat("/usr/bin/convert"); err == nil {
		cmd := []string{
			"convert", "-size", "256x256", "xc:"+color,
			"-fill", "white", "-font", "DejaVu-Sans-Bold", "-pointsize", "48",
			"-gravity", "center", "-draw", fmt.Sprintf("text 0,0 '%s'", symbol),
			iconPath,
		}
		
		// 简单执行命令
		process, err := os.StartProcess("/usr/bin/convert", cmd, &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		})
		if err == nil {
			process.Wait()
		}
	} else {
		// 创建简单的占位符文件
		os.WriteFile(iconPath, []byte("dummy icon"), 0644)
	}
}

func buildWithAppImageTool(appDir string) {
	// 检查appimagetool是否存在
	if _, err := os.Stat("/usr/bin/appimagetool"); os.IsNotExist(err) {
		fmt.Println("❌ appimagetool未安装，无法构建")
		fmt.Println("💡 手动构建命令:")
		fmt.Printf("   cd build\n")
		fmt.Printf("   ARCH=x86_64 appimagetool %s.AppDir %s\n", cfg.AppName, cfg.OutputFilename)
		return
	}

	fmt.Printf("🚀 构建AppImage: %s\n", cfg.OutputFilename)
	
	// 创建构建脚本
	buildScript := filepath.Join("build", "build.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
export ARCH=x86_64
appimagetool "%s" "%s"
`, appDir, filepath.Join("build", cfg.OutputFilename))
	os.WriteFile(buildScript, []byte(scriptContent), 0755)
	
	// 执行构建脚本
	cmd := []string{"/bin/bash", buildScript}
	process, err := os.StartProcess("/bin/bash", cmd, &os.ProcAttr{
		Dir:   "build",
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		fmt.Printf("❌ 构建失败: %v\n", err)
		return
	}
	process.Wait()

	// 移动到当前位置
	if fileExists(filepath.Join("build", cfg.OutputFilename)) {
		os.Rename(filepath.Join("build", cfg.OutputFilename), cfg.OutputFilename)
		os.Chmod(cfg.OutputFilename, 0755)
		fmt.Printf("✅ 构建完成: %s\n", filepath.Join(os.Getenv("PWD"), cfg.OutputFilename))
		fmt.Println("🔍 挂载后路径: /tmp/.mount_XXXX/game/")

		if cfg.PackageType == "wine" {
			wineArchiveDir := filepath.Join(cfg.WineArchiveBaseDir, cfg.AppName)
			fmt.Printf("📁 固定Archive目录: %s\n", wineArchiveDir)
			if cfg.WineSaveDir != "" {
				fmt.Printf("🎯 目录重定向模式: %s/\n", cfg.WineSaveDir)
			} else {
				fmt.Printf("🎯 自定义存档模式: %s (%d-%d)\n", cfg.SavePattern, cfg.SaveStart, cfg.SaveEnd)
			}
		} else {
			fmt.Printf("💾 统一存档位置: %s\n", filepath.Join(cfg.SaveBaseDir, cfg.AppName))
		}

		// 清理构建目录
		if cfg.ForceBuild || askForConfirmation("清理构建目录? [Y/n]: ", true) {
			os.RemoveAll("build")
			fmt.Println("🧹 构建目录已清理")
		}
	} else {
		fmt.Println("❌ 构建失败，文件未找到")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func getFirstTwoLetters(s string) string {
	runes := []rune(s)
	if len(runes) >= 2 {
		return strings.ToUpper(string(runes[0])) + strings.ToUpper(string(runes[1]))
	} else if len(runes) == 1 {
		return strings.ToUpper(string(runes[0]))
	}
	return "G"
}

func askForConfirmation(prompt string, defaultYes bool) bool {
	var response string
	fmt.Print(prompt)
	fmt.Scanln(&response)

	if response == "" {
		return defaultYes
	}

	response = strings.ToLower(response)
	return response == "y" || response == "yes" || response == "Y" || response == "YES"
}