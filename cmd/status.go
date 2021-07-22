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
)

var por bool

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "display current status",
	Long:  `display current status`,
	RunE: func(cmd *cobra.Command, args []string) error {

		rootPath, _ := os.Getwd()

		isLong := por

		w := os.Stdout
		if err := src.StartStatus(w, rootPath, isLong); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&por, "porcelain", "p", true, "display status in porcelain")
	rootCmd.AddCommand(statusCmd)
}
