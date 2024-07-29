package controller


import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

)


func (clctrl *ClusterController) TerraformPrep() error{

	path := clctrl.ProviderConfig.GitopsDir + "/terraform"
	err := filepath.Walk(path,detokenizeterraform(path,clctrl))
	if err != nil {
		return err
	}
	return nil
}

func detokenizeterraform(path string,clctrl *ClusterController) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string,fi os.FileInfo,err error) error{

		if fi.IsDir()  {
			return nil
		}

		matched,_ := filepath.Match("*",fi.Name())

		if matched {

			read,err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			newContents := string(read)
			newContents = strings.Replace(newContents,"<ADMIN_TEAM>",clctrl.AdminTeamName,-1)
			newContents = strings.Replace(newContents,"<DEVELOPER_TEAM>",clctrl.DeveloperTeamName,-1)
			newContents = strings.Replace(newContents,"<METAPHOR_REPO_NAME>",clctrl.MetaphorRepoName,-1)
			newContents = strings.Replace(newContents,"<GIT_REPO_NAME>",clctrl.GitopsRepoName,-1)

			err = ioutil.WriteFile(path,[]byte(newContents),0)
			if err != nil {
				return err
			}
			
		}
		return nil
	})

}