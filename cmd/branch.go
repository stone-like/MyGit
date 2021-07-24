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

var verbose bool
var delete bool
var force bool
var ForceDelete bool

// branchCmd represents the branch command
var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "create and list branch",
	Long:  `create and list branch`,
	Args:  cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {

		rootPath, _ := os.Getwd()
		bo := &src.BranchOption{
			HasV: verbose,
			HasD: delete,
			HasF: force,
		}
		w := os.Stdout
		if err := src.StartBranch(rootPath, args, bo, w); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVarP(&verbose, "verbose", "v", true, "verbose")
	diffCmd.Flags().BoolVarP(&delete, "delete", "d", true, "delete")
	diffCmd.Flags().BoolVarP(&force, "force", "f", true, "force")
	diffCmd.Flags().BoolVarP(&ForceDelete, "forceDelete", "D", true, "forceDelete")
	rootCmd.AddCommand(branchCmd)
}
