/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"mygit/src"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var message string

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := viper.GetString("name")
		email := viper.GetString("email")

		rootPath, _ := os.Getwd()
		w := os.Stdout

		if err := src.StartCommit(rootPath, name, email, message, w); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	commitCmd.Flags().StringVarP(&message, "message", "m", "", "commit message")
	rootCmd.AddCommand(commitCmd)
}
