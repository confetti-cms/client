package services

import (
	"encoding/base64"
	"encoding/json"
	"github.com/confetti-framework/errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"src/config"
	"strings"
	"unicode"
)

const hiddenDir = ".confetti"
const componentsDir = hiddenDir + "/Components"

// Actually, there should be another letter 'c' as the first letter here,
// but we don't consider it because it can be in lowercase or uppercase.
const componentConfigSuffix = "omponent.blade.php"
const componentClassSuffix = "omponent.class.php"

func UpsertHiddenComponentE(root string, file string, verbose bool) bool {
	err, done := UpsertHiddenComponent(root, file, verbose)
	if err != nil {
		println("Err UpsertHiddenComponentE:")
		println(err.Error())
		return false
	}
	return done
}

func UpsertHiddenComponent(root string, file string, verbose bool) (error, bool) {
	originFile := file
	// Check if it is a component generator
	if !strings.HasSuffix(file, componentConfigSuffix) {
		if !strings.HasSuffix(file, componentClassSuffix) {
			return nil, false
		}
		// If composer class has changed, handle it the same as the config file
		file = strings.Replace(file, componentClassSuffix, componentConfigSuffix, 1)
	}
	if verbose {
		println("Hidden component triggered by: " + originFile)
	}
	// Get content of component
	host := config.App.Host
	body, err := Send("http://api." + host +"/parser/source/components?file=/"+file, nil, http.MethodGet)
	if err != nil {
		return err, false
	}
	// Get file content from response
	contentsRaw := []map[string]string{}
	json.Unmarshal([]byte(body), &contentsRaw)
	if len(contentsRaw) == 0 {
		return errors.New("Can not upsert hidden component: file not found: " + file), false
	}
	contentRaw := contentsRaw[0]
	content64 := contentRaw["content"]
	name := contentRaw["name_class"]
	content, err := base64.StdEncoding.DecodeString(content64)
	if err != nil {
		return err, false
	}
	// Save hidden component
    target := path.Join(root, componentsDir, name+".php")
	err = os.MkdirAll(path.Dir(target), os.ModePerm)
	if err != nil {
		return err, false
	}
	f, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err, false
	}
	defer f.Close()
	_, err = f.WriteString(string(content))
	if err != nil {
		return err, false
	}
	if verbose {
		println("Hidden component saved: " + target)
	}
	return nil, true
}

func UpsertHiddenMap(root string, verbose bool) error {
	// Compose hidden Map component
    names, err := getComponentClassNamesByDirectory(path.Join(root, componentsDir))
	if err != nil {
		return err
	}
	content := getMapContent(names)
	// Save hidden Map component
    target := path.Join(root, componentsDir, "Map.php")
	err = os.MkdirAll(path.Dir(target), os.ModePerm)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if verbose {
		println("Hidden Map component saved: " + target)
	}
	return err
}

func getComponentClassNamesByDirectory(dir string) ([]string, error) {
	result := []string{}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return []string{}, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// We want to generate the map class, so ignore it
        if file.Name() == "Map.php" {
			continue
		}
        // Ignore helper files
        if !unicode.IsUpper(rune(file.Name()[0])) {
            continue
        }
        // We assume that the filename is equal to the classname.
        name := strings.TrimSuffix(file.Name(), ".php")
		result = append(result, name)
	}
	return result, nil
}

func getMapContent(classNames []string) string {
	contentRaw, err := config.Embed.Template.ReadFile("Map.php")
	if err != nil {
		panic(err)
	}
	content := string(contentRaw)
	for _, className := range classNames {
        functionName := getFunctionByClass(className)
		function := `    public function ` + functionName + `(string $key): ` + className + `
    {
        return new ` + className + `();
    }

//-> function`
		content = strings.Replace(content, "//-> function", function, 1)
	}
	content = strings.Replace(content, "\n//-> function", "", 1)
	return content
}

func getFunctionByClass(className string) string {
	if len(className) == 0 {
		return ""
	}
	// First character is lowercase
	char := []rune(className)
	char[0] = unicode.ToLower(char[0])
	name := string(char)
	// If class name is a preserved key, we suffix it before with _
	name = strings.TrimSuffix(name, "_")
	return name
}

func SaveStandardHiddenFiles(root string, verbose bool) error {
	// Get content of component
	host := config.App.Host
	body, err := Send("http://api." + host + "/parser/source/components/standard", nil, http.MethodGet)
	if err != nil {
		return err
	}
	// Get file content from response
	contentsRaw := []map[string]string{}
	json.Unmarshal([]byte(body), &contentsRaw)
	for _, contentRaw := range contentsRaw {
		content64 := contentRaw["content"]
		file := contentRaw["file"]
		content, err := base64.StdEncoding.DecodeString(content64)
		if err != nil {
			return err
		}
		// Save hidden component
		target := path.Join(root, hiddenDir, file)
		err = os.MkdirAll(path.Dir(target), os.ModePerm)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(string(content))
		if err != nil {
			return err
		}
		if verbose {
			println("Standard hidden component saved: " + target)
		}
	}
	return nil
}
