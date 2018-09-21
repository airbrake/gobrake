package gobrake

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("findGitDir", func() {
	It("returns git directory filepath", func() {
		workDir, _ := os.Getwd()
		gitDir := filepath.Join(workDir, ".git")
		tests := []struct {
			workDir string
			gitDir  string
			ok      bool
		}{
			{"", gitDir, true},
			{"abc", "", false},
			{workDir, gitDir, true},
			{filepath.Join(workDir, "internal"), gitDir, true},
			{filepath.Join(workDir, "internal", "lrucache"), gitDir, true},
			{filepath.Join(workDir, "abc"), "", false},
			{filepath.Dir(workDir), "", false},
		}

		for _, test := range tests {
			gitDir, ok := findGitDir(test.workDir)
			Expect(ok).To(Equal(test.ok))
			Expect(gitDir).To(Equal(test.gitDir))
		}
	})
})
