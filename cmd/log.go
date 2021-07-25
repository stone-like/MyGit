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

var abbrev bool
var oneline bool
var pretty string
var format string

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var hasAbbr bool
		var useFormat string
		if abbrev {
			hasAbbr = true
		}

		if pretty != "" {
			useFormat = pretty
		}

		if format != "" {
			useFormat = format
		}

		if oneline {
			hasAbbr = true
			useFormat = "oneline"
		}

		w := os.Stdout

		cur, err := os.Getwd()
		if err != nil {
			return err
		}

		o := &src.LogOption{
			IsAbbrev: hasAbbr,
			Format:   useFormat,
		}

		err = src.StartLog(cur, args, o, w)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVarP(&abbrev, "abbrev-commit", "ac", true, "abbrev")
	diffCmd.Flags().BoolVarP(&oneline, "oneline", "on", true, "oneline")
	commitCmd.Flags().StringVarP(&pretty, "pretty", "p", "", "pretty")
	commitCmd.Flags().StringVarP(&format, "format", "f", "", "format")
	rootCmd.AddCommand(logCmd)
}
