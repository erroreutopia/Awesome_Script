package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 配置结构体
type Config struct {
	GameSourceDir     string
	AppName           string
	IconPath          string
	PackageType       string
	WineExec          string
	WineCmd           string
	WineSaveDir       string
	RootSaveFiles     []string // 根目录存档文件
	SavePattern       string
	SaveStart         int
	SaveEnd           int
	AutoBuild         bool
	ForceBuild        bool
	OutputFilename    string
	NWJSPath          string
	SaveBaseDir       string
	WineArchiveBaseDir string
}

var cfg Config

func main() {
	// 初始化默认配置
	cfg = Config{
		WineCmd:           "proton-ge",
		SavePattern:       "Save%d",
		SaveStart:         1,
		SaveEnd:           10,
		NWJSPath:          filepath.Join(os.Getenv("HOME"), "App/nwjs-sdk/nw"),
		SaveBaseDir:       filepath.Join(os.Getenv("HOME"), "Game/HTMLGame/NWJS/SAVE"),
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
	rootSave := flag.String("root-save", "", "根目录存档文件 (逗号分隔)")
	rootSaveLong := flag.String("root-save-files", "", "根目录存档文件 (逗号分隔)")
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

	// 处理根目录存档文件
	rootSaveFiles := *rootSave
	if rootSaveFiles == "" {
		rootSaveFiles = *rootSaveLong
	}
	if rootSaveFiles != "" {
		cfg.RootSaveFiles = strings.Split(rootSaveFiles, ",")
		for i := range cfg.RootSaveFiles {
			cfg.RootSaveFiles[i] = strings.TrimSpace(cfg.RootSaveFiles[i])
		}
	}

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
	fmt.Println("  --root-save FILES      根目录存档文件 (逗号分隔，例如: save.dat,config.ini)")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  # 问卷式模式 (无参数)")
	fmt.Println("  agamepack")
	fmt.Println("")
	fmt.Println("  # 指定参数 (根目录存档)")
	fmt.Println("  agamepack -r \"old_game\" -n \"OldGame\" --wine-exec \"game.exe\" \\")
	fmt.Println("    --root-save \"save.dat,config.ini\" --build -y")
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

	// 3. 图标文件
	fmt.Print("自定义图标文件 (留空使用默认生成): ")
	var iconPath string
	fmt.Scanln(&iconPath)
	cfg.IconPath = iconPath
	if cfg.IconPath != "" {
		absIconPath, err := filepath.Abs(cfg.IconPath)
		if err == nil {
			cfg.IconPath = absIconPath
		}
		if _, err := os.Stat(cfg.IconPath); os.IsNotExist(err) {
			fmt.Printf("⚠️  图标文件不存在: %s，将使用默认生成\n", cfg.IconPath)
			cfg.IconPath = ""
		} else {
			fmt.Printf("✅ 使用自定义图标: %s\n", cfg.IconPath)
		}
	}

	// 4. 游戏类型
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
			setupRootSaveFilesInteractive()
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

func setupRootSaveFilesInteractive() {
	fmt.Println("")
	fmt.Println("🔍 检测根目录存档文件...")
	rootFiles := findRootSaveFiles(cfg.GameSourceDir)
	if len(rootFiles) > 0 {
		fmt.Println("检测到可能的根目录存档文件:")
		for i, file := range rootFiles {
			fmt.Printf("  %d. %s\n", i+1, file)
		}
		var choices string
		fmt.Print("选择要重定向的文件 (例如: 1,2,3 或 0 跳过): ")
		fmt.Scanln(&choices)
		if choices != "0" && choices != "" {
			selected := strings.Split(choices, ",")
			for _, choice := range selected {
				choice = strings.TrimSpace(choice)
				index, err := strconv.Atoi(choice)
				if err == nil && index > 0 && index <= len(rootFiles) {
					cfg.RootSaveFiles = append(cfg.RootSaveFiles, rootFiles[index-1])
				}
			}
		}
	}

	if len(cfg.RootSaveFiles) == 0 {
		var manualFiles string
		fmt.Print("手动指定根目录存档文件 (逗号分隔，例如: save.dat,config.ini，留空跳过): ")
		fmt.Scanln(&manualFiles)
		if manualFiles != "" {
			files := strings.Split(manualFiles, ",")
			for _, file := range files {
				file = strings.TrimSpace(file)
				if file != "" {
					cfg.RootSaveFiles = append(cfg.RootSaveFiles, file)
				}
			}
		}
	}

	if len(cfg.RootSaveFiles) > 0 {
		fmt.Printf("✅ 选择根目录存档文件: %v\n", cfg.RootSaveFiles)
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
	fmt.Println("2. 根目录存档文件 (例如: save.dat, config.ini)")
	fmt.Println("3. 自定义文件模式 (Save01, Save02...)")
	fmt.Println("4. 混合模式 (目录 + 根目录文件)")
	var choice string
	fmt.Print("选择存档方式 (推荐 2 或 4): ")
	fmt.Scanln(&choice)
	if choice == "" {
		choice = "2"
	}
	switch choice {
	case "1":
		setupWineSaveDirInteractive()
	case "2":
		setupRootSaveFilesInteractive()
	case "3":
		setupCustomSavePatternInteractive()
	case "4":
		setupWineSaveDirInteractive()
		setupRootSaveFilesInteractive()
	default:
		fmt.Println("❌ 无效选择，使用默认: 根目录存档文件")
		setupRootSaveFilesInteractive()
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
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				return r
			}
			return -1
		}, cfg.AppName)
		if cleanName == "" {
			cleanName = "Game"
		}
		// 确保有.AppImage后缀
		if !strings.HasSuffix(cleanName, ".AppImage") {
			cleanName += ".AppImage"
		}
		cfg.OutputFilename = cleanName
		fmt.Printf("📝 使用目录名作为默认文件名: %s\n", cfg.OutputFilename)
	} else {
		// 确保有.AppImage后缀
		if !strings.HasSuffix(cfg.OutputFilename, ".AppImage") {
			cfg.OutputFilename += ".AppImage"
		}
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
	
	// 清理旧的构建目录
	os.RemoveAll("build")
	
	// 确保目标目录存在
	os.MkdirAll(gameSubDir, 0755)
	
	// 复制游戏文件
	if err := copyDir(cfg.GameSourceDir, gameSubDir); err != nil {
		fmt.Printf("❌ 复制文件失败: %v\n", err)
		os.Exit(1)
	}

	// 存档处理 - 仅在game/目录内创建符号链接
	if cfg.PackageType == "wine" {
		fmt.Println("🎯 Wine应用: 存档处理")
		wineArchiveDir := filepath.Join(cfg.WineArchiveBaseDir, cfg.AppName)
		os.MkdirAll(wineArchiveDir, 0755)
		fmt.Printf("📁 固定Archive目录: %s\n", wineArchiveDir)
		
		// 1. 目录重定向模式
		if cfg.WineSaveDir != "" {
			fmt.Printf("🔗 目录重定向模式: %s/\n", cfg.WineSaveDir)
			wineSavePath := filepath.Join(gameSubDir, cfg.WineSaveDir)
			targetSavePath := filepath.Join(wineArchiveDir, cfg.WineSaveDir)
			
			// 确保目标目录存在
			os.MkdirAll(targetSavePath, 0755)
			
			// 安全地创建符号链接 - 先移除目标（如果存在）
			if _, err := os.Stat(wineSavePath); err == nil {
				if isDir(wineSavePath) {
					os.RemoveAll(wineSavePath)
				} else {
					os.Remove(wineSavePath)
				}
			}
			
			// 创建符号链接
			if err := os.Symlink(targetSavePath, wineSavePath); err != nil {
				fmt.Printf("⚠️  创建符号链接失败: %v\n", err)
				
				// 如果符号链接创建失败，尝试直接复制内容
				fmt.Println("🔄 尝试直接复制存档文件...")
				if err := copyDirIfExists(targetSavePath, wineSavePath); err != nil {
					fmt.Printf("⚠️  复制存档文件失败，但将继续: %v\n", err)
				}
			} else {
				fmt.Printf("✅ 目录重定向完成: %s -> %s\n", wineSavePath, targetSavePath)
			}
		}
		
		// 2. 根目录存档文件
		if len(cfg.RootSaveFiles) > 0 {
			fmt.Printf("🔗 根目录存档文件: %v\n", cfg.RootSaveFiles)
			totalLinks := 0
			for _, filename := range cfg.RootSaveFiles {
				sourceFile := filepath.Join(gameSubDir, filename)
				targetFile := filepath.Join(wineArchiveDir, filename)
				
				// 确保目标目录存在
				os.MkdirAll(filepath.Dir(targetFile), 0755)
				
				// 安全地创建符号链接
				if _, err := os.Stat(sourceFile); err == nil {
					if isDir(sourceFile) {
						os.RemoveAll(sourceFile)
					} else {
						os.Remove(sourceFile)
					}
				}
				
				if err := os.Symlink(targetFile, sourceFile); err != nil {
					fmt.Printf("⚠️  创建符号链接失败: %v\n", err)
					
					// 尝试复制文件内容
					if _, err := os.Stat(targetFile); err == nil {
						copyFile(targetFile, sourceFile)
					}
				} else {
					fmt.Printf("✅ 根目录存档链接: %s -> %s\n", sourceFile, targetFile)
					totalLinks++
				}
			}
			fmt.Printf("✅ 总共创建 %d 个根目录存档链接\n", totalLinks)
		}
		
		// 3. 自定义文件模式 (如果没有其他存档设置)
		if cfg.WineSaveDir == "" && len(cfg.RootSaveFiles) == 0 {
			fmt.Printf("🔗 创建自定义存档链接: %s (%d to %d)\n",
				cfg.SavePattern, cfg.SaveStart, cfg.SaveEnd)
			totalLinks := 0
			for i := cfg.SaveStart; i <= cfg.SaveEnd; i++ {
				filename := fmt.Sprintf(cfg.SavePattern, i)
				sourceFile := filepath.Join(gameSubDir, filename)
				targetFile := filepath.Join(wineArchiveDir, filename)
				
				// 确保目标目录存在
				os.MkdirAll(filepath.Dir(targetFile), 0755)
				
				// 安全地创建符号链接
				if _, err := os.Stat(sourceFile); err == nil {
					os.Remove(sourceFile)
				}
				
				if err := os.Symlink(targetFile, sourceFile); err != nil {
					fmt.Printf("⚠️  创建符号链接失败: %v\n", err)
				} else {
					fmt.Printf("✅ 自定义存档链接: %s -> %s\n", sourceFile, targetFile)
					totalLinks++
				}
			}
			fmt.Printf("✅ 总共创建 %d 个自定义存档链接\n", totalLinks)
		}
	} else {
		// NW.js: 只在game/目录内创建符号链接
		gameSaveDir := filepath.Join(cfg.SaveBaseDir, cfg.AppName)
		os.MkdirAll(gameSaveDir, 0755)
		
		// 只在game/目录内创建链接
		os.MkdirAll(filepath.Join(gameSubDir, "save"), 0755)
		createLink(gameSaveDir, filepath.Join(gameSubDir, "save"))
		
		// 为www目录创建链接（如果存在）
		wwwDir := filepath.Join(gameSubDir, "www")
		if dirExists(wwwDir) {
			os.MkdirAll(filepath.Join(wwwDir, "save"), 0755)
			createLink(gameSaveDir, filepath.Join(wwwDir, "save"))
		}
		
		// 根目录存档文件
		if len(cfg.RootSaveFiles) > 0 {
			fmt.Printf("🔗 NW.js根目录存档文件: %v\n", cfg.RootSaveFiles)
			totalLinks := 0
			for _, filename := range cfg.RootSaveFiles {
				sourceFile := filepath.Join(gameSubDir, filename)
				targetFile := filepath.Join(gameSaveDir, filename)
				
				// 确保目标目录存在
				os.MkdirAll(filepath.Dir(targetFile), 0755)
				
				// 安全地创建符号链接
				if _, err := os.Stat(sourceFile); err == nil {
					os.Remove(sourceFile)
				}
				
				if err := os.Symlink(targetFile, sourceFile); err != nil {
					fmt.Printf("⚠️  创建符号链接失败: %v\n", err)
				} else {
					fmt.Printf("✅ 根目录存档链接: %s -> %s\n", sourceFile, targetFile)
					totalLinks++
				}
			}
			fmt.Printf("✅ 总共创建 %d 个根目录存档链接\n", totalLinks)
		}
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
		
		// 构建成功后清理构建目录
		if cfg.ForceBuild || askForConfirmation("清理构建目录? [Y/n]: ", true) {
			os.RemoveAll("build")
			fmt.Println("🧹 构建目录已清理")
		}
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
			if dirName == "save" || dirName == "saves" || dirName == "data" || dirName == "userdata" || dirName == "mcsc" {
				relPath, _ := filepath.Rel(dir, path)
				saveDirs = append(saveDirs, relPath)
			}
		}
		return nil
	})
	return saveDirs
}

func findRootSaveFiles(dir string) []string {
	var saveFiles []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			filename := strings.ToLower(d.Name())
			// 常见的存档文件扩展名
			extensions := []string{".sav", ".save", ".dat", ".ini", ".cfg", ".conf", ".json", ".bin", ".srm"}
			for _, ext := range extensions {
				if strings.HasSuffix(filename, ext) {
					relPath, _ := filepath.Rel(dir, path)
					saveFiles = append(saveFiles, relPath)
					break
				}
			}
		}
		return nil
	})
	return saveFiles
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过符号链接
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
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

func copyDirIfExists(src string, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	return copyDir(src, dst)
}

func copyFile(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func dirExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func createLink(target string, link string) {
	// 确保目标目录存在
	os.MkdirAll(target, 0755)
	
	// 安全地创建符号链接
	if _, err := os.Stat(link); err == nil {
		if isDir(link) {
			os.RemoveAll(link)
		} else {
			os.Remove(link)
		}
	}
	
	os.MkdirAll(filepath.Dir(link), 0755)
	if err := os.Symlink(target, link); err != nil {
		fmt.Printf("⚠️  创建符号链接失败: %v\n", err)
	}
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
# 确保存档目录存在
if [ -n "%s" ]; then
    mkdir -p "${WINE_ARCHIVE_DIR}/%s" 2>/dev/null || true
fi
exec "${WINE_CMD}" "%s"
`, cfg.AppName, cfg.WineArchiveBaseDir, cfg.WineCmd, cfg.WineSaveDir, cfg.WineSaveDir, cfg.WineExec)
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
	
	err := os.WriteFile(appRunPath, []byte(content), 0755)
	if err != nil {
		fmt.Printf("❌ 创建AppRun失败: %v\n", err)
		os.Exit(1)
	}
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
`, cfg.AppName, cfg.AppName)
	
	err := os.WriteFile(desktopPath, []byte(content), 0644)
	if err != nil {
		fmt.Printf("❌ 创建.desktop文件失败: %v\n", err)
		os.Exit(1)
	}
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
	
	// 检查ImageMagick命令
	convertCmd := "convert"
	if _, err := exec.LookPath(convertCmd); err != nil {
		// 尝试 magick
		convertCmd = "magick"
		if _, err := exec.LookPath(convertCmd); err != nil {
			convertCmd = ""
		}
	}
	
	// 生成图标
	if convertCmd != "" {
		var cmdArgs []string
		if convertCmd == "magick" {
			// IMv7 语法: magick convert [options]
			cmdArgs = []string{
				"convert", "-size", "256x256", "xc:"+color,
				"-fill", "white", "-font", "DejaVu-Sans-Bold", "-pointsize", "48",
				"-gravity", "center", "-draw", fmt.Sprintf("text 0,0 '%s'", symbol),
				iconPath,
			}
		} else {
			// IMv6 语法: convert [options]
			cmdArgs = []string{
				"-size", "256x256", "xc:"+color,
				"-fill", "white", "-font", "DejaVu-Sans-Bold", "-pointsize", "48",
				"-gravity", "center", "-draw", fmt.Sprintf("text 0,0 '%s'", symbol),
				iconPath,
			}
		}
		// 执行命令
		var cmd *exec.Cmd
		if convertCmd == "magick" {
			cmd = exec.Command("magick", cmdArgs...)
		} else {
			cmd = exec.Command("convert", cmdArgs...)
		}
		err := cmd.Run()
		if err == nil && fileExists(iconPath) {
			return
		}
	}
	
	// 创建简单的占位符文件
	err := os.WriteFile(iconPath, []byte("dummy icon"), 0644)
	if err == nil {
		fmt.Println("⚠️  无法生成图标，使用占位符")
	} else {
		fmt.Printf("❌ 创建图标文件失败: %v\n", err)
	}
}

func buildWithAppImageTool(appDir string) {
	// 检查appimagetool是否存在
	appimagetoolPath := "appimagetool"
	if _, err := exec.LookPath(appimagetoolPath); err != nil {
		// 尝试其他路径
		for _, path := range []string{"/usr/bin/appimagetool", "/usr/local/bin/appimagetool"} {
			if _, err := os.Stat(path); err == nil {
				appimagetoolPath = path
				break
			}
		}
	}
	if _, err := exec.LookPath(appimagetoolPath); err != nil {
		fmt.Printf("❌ appimagetool未安装: %v\n", err)
		fmt.Println("💡 安装命令 (Debian/Ubuntu): sudo apt-get install appimagetool")
		fmt.Println("💡 安装命令 (Arch Linux): sudo pacman -S appimagetool")
		fmt.Println("💡 手动构建命令:")
		fmt.Printf("   cd build\n")
		fmt.Printf("   ARCH=x86_64 %s \"%s\" \"%s\"\n", appimagetoolPath, filepath.Base(appDir), cfg.OutputFilename)
		return
	}
	
	buildOutput := filepath.Join("build", cfg.OutputFilename)
	fmt.Printf("🚀 构建AppImage: %s\n", cfg.OutputFilename)
	fmt.Printf("   📁 源目录: %s\n", appDir)
	fmt.Printf("   🎯 输出: %s\n", buildOutput)
	
	// 确保输出目录存在
	os.MkdirAll(filepath.Dir(buildOutput), 0755)
	
	// 设置环境变量
	env := os.Environ()
	env = append(env, "ARCH=x86_64")
	env = append(env, "APPIMAGE_EXTRACT_AND_RUN=1") // 关键：避免权限问题
	
	// 关键修正：使用相对路径
	appDirName := filepath.Base(appDir) // 只取目录名，不包含build/
	
	// 确保构建目录存在
	os.MkdirAll("build", 0755)
	
	// 在build目录中执行appimagetool
	cmd := exec.Command(appimagetoolPath, appDirName, cfg.OutputFilename)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "build"  // 在build目录中执行
	fmt.Printf("🔧 工作目录: %s\n", cmd.Dir)
	fmt.Printf("🔧 命令: %s %s %s\n", appimagetoolPath, appDirName, cfg.OutputFilename)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("❌ 构建失败: %v\n", err)
		
		// 检查构建目录
		fmt.Println("🔍 检查构建目录内容:")
		files, err := os.ReadDir("build")
		if err == nil {
			for _, file := range files {
				fmt.Printf("   - %s\n", file.Name())
			}
		} else {
			fmt.Printf("   ❌ 无法读取build目录: %v\n", err)
		}
		
		// 检查源目录是否存在
		if info, err := os.Stat(appDir); os.IsNotExist(err) {
			fmt.Printf("❌ 源目录不存在: %s\n", appDir)
		} else if err == nil {
			fmt.Printf("✅ 源目录存在: %s\n", appDir)
			fmt.Printf("   📁 大小: %d bytes\n", info.Size())
		} else {
			fmt.Printf("❌ 检查源目录失败: %v\n", err)
		}
		
		// 检查输出文件是否存在
		if fileExists(buildOutput) {
			fmt.Printf("✅ 输出文件存在: %s\n", buildOutput)
		} else {
			fmt.Printf("❌ 输出文件不存在: %s\n", buildOutput)
		}
		
		// 提示手动构建
		fmt.Println("\n💡 手动构建命令:")
		fmt.Printf("   cd build\n")
		fmt.Printf("   ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 %s \"%s\" \"%s\"\n", 
			appimagetoolPath, appDirName, cfg.OutputFilename)
		return
	}
	
	// 检查构建结果
	if fileExists(buildOutput) {
		// 移动到当前位置
		currentPath := filepath.Join(".", cfg.OutputFilename)
		err := os.Rename(buildOutput, currentPath)
		if err != nil {
			fmt.Printf("❌ 移动文件失败: %v\n", err)
			
			// 尝试复制
			fmt.Println("🔄 尝试复制文件...")
			if copyFile(buildOutput, currentPath) == nil {
				os.Remove(buildOutput)
				fmt.Println("✅ 文件复制成功")
			} else {
				fmt.Println("❌ 文件复制也失败")
				return
			}
		}
		
		// 设置执行权限
		os.Chmod(currentPath, 0755)
		fmt.Printf("✅ 构建完成: %s\n", filepath.Join(os.Getenv("PWD"), cfg.OutputFilename))
		fmt.Println("🔍 挂载后路径: /tmp/.mount_XXXX/game/")
		
		if cfg.PackageType == "wine" {
			wineArchiveDir := filepath.Join(cfg.WineArchiveBaseDir, cfg.AppName)
			fmt.Printf("📁 固定Archive目录: %s\n", wineArchiveDir)
			if cfg.WineSaveDir != "" {
				fmt.Printf("🎯 目录重定向模式: %s/\n", cfg.WineSaveDir)
			}
			if len(cfg.RootSaveFiles) > 0 {
				fmt.Printf("🎯 根目录存档文件: %v\n", cfg.RootSaveFiles)
			}
			if cfg.WineSaveDir == "" && len(cfg.RootSaveFiles) == 0 {
				fmt.Printf("🎯 自定义存档模式: %s (%d-%d)\n", cfg.SavePattern, cfg.SaveStart, cfg.SaveEnd)
			}
		} else {
			fmt.Printf("💾 统一存档位置: %s\n", filepath.Join(cfg.SaveBaseDir, cfg.AppName))
			if len(cfg.RootSaveFiles) > 0 {
				fmt.Printf("🎯 根目录存档文件: %v\n", cfg.RootSaveFiles)
			}
		}
	} else {
		fmt.Println("❌ 构建失败，输出文件未找到")
		
		// 检查构建目录
		fmt.Println("🔍 检查构建目录内容:")
		files, err := os.ReadDir("build")
		if err == nil {
			for _, file := range files {
				fmt.Printf("   - %s\n", file.Name())
			}
		} else {
			fmt.Printf("   ❌ 无法读取build目录: %v\n", err)
		}
		
		// 检查源目录
		if info, err := os.Stat(appDir); os.IsNotExist(err) {
			fmt.Printf("❌ 源目录不存在: %s\n", appDir)
		} else if err == nil {
			fmt.Printf("✅ 源目录存在: %s\n", appDir)
			fmt.Printf("   📁 大小: %d bytes\n", info.Size())
		} else {
			fmt.Printf("❌ 检查源目录失败: %v\n", err)
		}
		
		// 检查输出文件
		if fileExists(buildOutput) {
			fmt.Printf("✅ 输出文件存在: %s\n", buildOutput)
		} else {
			fmt.Printf("❌ 输出文件不存在: %s\n", buildOutput)
		}
		
		// 提示手动构建
		fmt.Println("\n💡 手动构建命令:")
		fmt.Printf("   cd build\n")
		fmt.Printf("   ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 %s \"%s\" \"%s\"\n", 
			appimagetoolPath, appDirName, cfg.OutputFilename)
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