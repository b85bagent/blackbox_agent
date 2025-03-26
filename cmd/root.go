package cmd

import (
	"blackbox_agent/pkg/autoload"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func Run() {
	var targetFile, configFile, blackboxFile string

	rootCmd := &cobra.Command{
		Use:   "blackbox_agent",
		Short: "Blackbox Agent Application",
		Long:  "Blackbox Agent using Cli for setting yaml configuration",
		Run: func(cmd *cobra.Command, args []string) {
			// 在這裡添加你的應用程式邏輯
			autoload.AutoLoader(configFile, targetFile, blackboxFile)
		},
	}

	// 添加 YAML 環境檔案選項
	rootCmd.PersistentFlags().StringVarP(&targetFile, "target", "t", "target.yaml", "指定要載入的 Target YAML 檔案 (檔案請以target開頭)")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "指定要載入的設定 Config YAML 檔案 (檔案請以config開頭)")
	rootCmd.PersistentFlags().StringVarP(&blackboxFile, "blackbox", "b", "blackbox.yaml", "指定要載入的設定 Blackbox YAML 檔案 (檔案請以blackbox開頭)")

	// 在執行根命令之前的預處理邏輯
	rootCmd.PreRun = func(cmd *cobra.Command, args []string) {
		log.Printf("使用的 Config 檔案: %s / 使用的 Target 檔案: %s / 使用的 Blackbox 檔案: %s\n", configFile, targetFile, blackboxFile)
	}

	// 執行根命令
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
