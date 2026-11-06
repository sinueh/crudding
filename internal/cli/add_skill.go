package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// DefaultSkillURL é o endereço padrão do arquivo AGENTS.md na raiz do repositório remoto.
const DefaultSkillURL = "https://raw.githubusercontent.com/sinueh/crudding/main/AGENTS.md"

// SkillDestPath é o caminho relativo onde a skill do crudding é instalada no projeto.
var SkillDestPath = filepath.Join(".agents", "skills", "crudding", "SKILL.md")

const maxSkillBytes = 1 << 20 // 1 MiB

// AddSkillUsage gera o texto de ajuda do comando "crudding add skill".
func AddSkillUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Baixa o arquivo AGENTS.md da raiz do repositório do crudding no GitHub e\n")
	b.WriteString("  salva em .agents/skills/crudding/SKILL.md no diretório atual, instalando\n")
	b.WriteString("  ou atualizando as instruções para agentes de IA usarem o crudding.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding add skill\n")
	return b.String()
}

func skillSourceURL() string {
	if u := strings.TrimSpace(os.Getenv("CRUDDING_SKILL_URL")); u != "" {
		return u
	}
	return DefaultSkillURL
}

func fetchSkillContent(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("baixar skill de %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("baixar skill de %s: status HTTP %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSkillBytes))
	if err != nil {
		return nil, fmt.Errorf("ler conteúdo da skill: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("conteúdo da skill vazio em %s", url)
	}
	return data, nil
}

func runAddSkill(args []string, stdout, _ io.Writer) error {
	if slices.ContainsFunc(args, func(token string) bool { return slices.Contains(helpFlags, token) }) {
		fmt.Fprint(stdout, AddSkillUsage())
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("argumentos inesperados: %s", strings.Join(args, " "))
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}

	url := skillSourceURL()
	content, err := fetchSkillContent(url)
	if err != nil {
		return err
	}

	destPath := filepath.Join(workingDirectory, SkillDestPath)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("criar diretório da skill: %w", err)
	}
	if err := os.WriteFile(destPath, content, 0o644); err != nil {
		return fmt.Errorf("gravar skill em %s: %w", destPath, err)
	}

	fmt.Fprintf(stdout, "Skill do crudding instalada em %s\n", destPath)
	return nil
}
