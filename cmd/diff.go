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

var cached bool
var base string
var theirs string
var ours string

func GetStage() int {
	if base != "" {
		return 1
	}

	if theirs != "" {
		return 2
	}

	if ours != "" {
		return 3
	}

	return 0
}

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "display diff",
	Long:  `display diff`,
	RunE: func(cmd *cobra.Command, args []string) error {

		rootPath, _ := os.Getwd()

		o := &src.DiffOption{
			Cached: cached,
			Stage:  GetStage(),
		}

		w := os.Stdout
		if err := src.StartDiff(w, rootPath, o); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVarP(&cached, "cached", "c", true, "display index<->commit diff")
	diffCmd.Flags().StringVarP(&base, "base", "b", "", "diff base")
	diffCmd.Flags().StringVarP(&theirs, "theirs", "t", "", "diff theirs")
	diffCmd.Flags().StringVarP(&ours, "ours", "o", "", "diff ours")
	rootCmd.AddCommand(diffCmd)
}
