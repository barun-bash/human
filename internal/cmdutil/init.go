package cmdutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// AppType describes a project starter template type.
type AppType struct {
	Name        string // display name
	Key         string // selection key
	Description string // one-line description
	ExampleDir  string // if non-empty, copy from examples/<ExampleDir>/app.human
}

// AvailableAppTypes returns the built-in app type templates.
func AvailableAppTypes() []AppType {
	return []AppType{
		{Name: "Auth app", Key: "auth", Description: "Sign in, sign up, forgot password, profile"},
		{Name: "CRUD app", Key: "crud", Description: "List, detail, create, edit, delete pages"},
		{Name: "Dashboard", Key: "dashboard", Description: "Analytics, charts, metrics, admin panel", ExampleDir: "saas"},
		{Name: "E-commerce", Key: "ecommerce", Description: "Products, cart, checkout, orders", ExampleDir: "ecommerce"},
		{Name: "Blog/CMS", Key: "blog", Description: "Posts, categories, comments, editor", ExampleDir: "blog"},
		{Name: "SaaS starter", Key: "saas", Description: "Auth + dashboard + settings + billing", ExampleDir: "saas"},
		{Name: "Empty", Key: "empty", Description: "Blank .human file with just the skeleton"},
		{Name: "Custom (LLM)", Key: "custom", Description: "Describe freely, requires /connect"},
	}
}

// DesignSystem describes a frontend design system option.
type DesignSystem struct {
	Name    string // display name
	Key     string // config value for the .human file
	Package string // npm package name (for reference)
}

// AvailableDesignSystems returns the built-in design system choices.
func AvailableDesignSystems() []DesignSystem {
	return []DesignSystem{
		{Name: "Shadcn/ui", Key: "Shadcn", Package: "shadcn/ui"},
		{Name: "Tailwind only", Key: "Tailwind", Package: "tailwindcss"},
		{Name: "Material UI", Key: "Material", Package: "@mui/material"},
		{Name: "Chakra UI", Key: "Chakra", Package: "@chakra-ui/react"},
		{Name: "Ant Design", Key: "Ant Design", Package: "antd"},
		{Name: "Bootstrap", Key: "Bootstrap", Package: "bootstrap"},
	}
}

// InitProject scaffolds a new Human project via an interactive wizard.
// Prompts for app type, entity/fields (for CRUD), stack, and design system.
// Creates the project directory and writes app.human, HUMAN.md, and .gitignore.
// Returns the path to the generated .human file.
func InitProject(name string, in io.Reader, out io.Writer) (string, error) {
	if name == "" {
		fmt.Fprintf(out, "App name: ")
		scanner := bufio.NewScanner(in)
		if scanner.Scan() {
			name = strings.TrimSpace(scanner.Text())
		}
		if name == "" {
			dir, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("could not determine current directory")
			}
			name = filepath.Base(dir)
		}
	}

	scanner := bufio.NewScanner(in)

	fmt.Fprintln(out, "\nCreate a new Human project (no LLM required)")
	fmt.Fprintln(out)

	// Step 1: App type selection.
	appTypes := AvailableAppTypes()
	fmt.Fprintln(out, "Select app type:")
	for i, t := range appTypes {
		fmt.Fprintf(out, "  %d. %-16s — %s\n", i+1, t.Name, t.Description)
	}
	fmt.Fprintln(out)

	typeIdx := 0
	fmt.Fprintf(out, "Choice [1]: ")
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(appTypes) {
				typeIdx = n - 1
			} else {
				for i, t := range appTypes {
					if strings.EqualFold(input, t.Key) || strings.EqualFold(input, t.Name) {
						typeIdx = i
						break
					}
				}
			}
		}
	}
	chosen := appTypes[typeIdx]

	// If "Custom (LLM)", return a signal that the caller should use /ask instead.
	if chosen.Key == "custom" {
		return "", fmt.Errorf("custom: use /ask to describe your app with LLM assistance")
	}

	// Step 2: For CRUD apps, ask about entity and fields.
	entityName := ""
	entityFields := []entityField{}
	if chosen.Key == "crud" {
		entityName, entityFields = promptEntityFields(scanner, out)
	}

	// Step 3: Try loading from examples directory for pre-built templates.
	var content string
	if chosen.ExampleDir != "" {
		loaded, err := loadExampleTemplate(chosen.ExampleDir, name)
		if err == nil {
			content = loaded
		}
	}

	// Step 4: If no example loaded, build from wizard.
	if content == "" {
		frontend := Prompt(scanner, out, "Frontend", []string{"React", "Vue", "Angular", "Svelte", "None"}, "React")

		// Design system (only if frontend is selected).
		designSystem := ""
		if !strings.EqualFold(frontend, "None") {
			designSystem = promptDesignSystem(scanner, out)
		}

		backend := Prompt(scanner, out, "Backend", []string{"Node", "Python", "Go"}, "Node")
		database := Prompt(scanner, out, "Database", []string{"PostgreSQL", "MySQL", "SQLite"}, "PostgreSQL")

		content = generateFromType(name, chosen.Key, frontend, backend, database, designSystem, entityName, entityFields)
	}

	// Create project directory.
	if err := os.MkdirAll(name, 0755); err != nil {
		return "", fmt.Errorf("could not create directory %s: %w", name, err)
	}

	// Write app.human.
	outPath := filepath.Join(name, "app.human")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("could not write %s: %w", outPath, err)
	}

	// Write HUMAN.md.
	humanMD := generateHumanMD(name, chosen)
	humanMDPath := filepath.Join(name, "HUMAN.md")
	os.WriteFile(humanMDPath, []byte(humanMD), 0644) // non-fatal

	// Write .gitignore.
	gitignore := generateGitignore()
	gitignorePath := filepath.Join(name, ".gitignore")
	os.WriteFile(gitignorePath, []byte(gitignore), 0644) // non-fatal

	// Summary.
	fmt.Fprintln(out)
	summary := summarizeContent(content)
	if summary != "" {
		fmt.Fprintln(out, summary)
	}

	return outPath, nil
}

// ── Entity/Fields Prompt ──

type entityField struct {
	Name string
	Type string
}

var defaultEntityFields = []entityField{
	{Name: "title", Type: "text"},
	{Name: "description", Type: "text"},
	{Name: "status", Type: "text"},
	{Name: "due_date", Type: "date"},
}

func promptEntityFields(scanner *bufio.Scanner, out io.Writer) (string, []entityField) {
	fmt.Fprintf(out, "\nWhat are you managing? (e.g., tasks, products, recipes): ")
	entityName := "Item"
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			entityName = capitalize(input)
		}
	}

	// Show default fields.
	fmt.Fprintln(out, "  Default fields: title (text), description (text), status (text), due_date (date)")
	fmt.Fprintf(out, "  Add more? (comma-separated, or Enter for defaults): ")

	fields := make([]entityField, len(defaultEntityFields))
	copy(fields, defaultEntityFields)

	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			for _, f := range strings.Split(input, ",") {
				f = strings.TrimSpace(f)
				if f != "" {
					fields = append(fields, entityField{Name: f, Type: "text"})
				}
			}
		}
	}

	return entityName, fields
}

// ── Design System Prompt ──

func promptDesignSystem(scanner *bufio.Scanner, out io.Writer) string {
	systems := AvailableDesignSystems()
	fmt.Fprintln(out, "\nSelect design system:")
	for i, ds := range systems {
		rec := ""
		if i == 0 {
			rec = " (recommended)"
		}
		fmt.Fprintf(out, "  %d. %s%s\n", i+1, ds.Name, rec)
	}
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Choice [1]: ")
	dsIdx := 0
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(systems) {
				dsIdx = n - 1
			}
		}
	}
	return systems[dsIdx].Key
}

// ── Template Generators ──

func generateFromType(name, appType, frontend, backend, database, designSystem, entityName string, fields []entityField) string {
	switch appType {
	case "auth":
		return generateAuthApp(name, frontend, backend, database, designSystem)
	case "crud":
		return generateCRUDApp(name, frontend, backend, database, designSystem, entityName, fields)
	case "dashboard":
		return generateDashboardApp(name, frontend, backend, database, designSystem)
	case "ecommerce":
		return generateEcommerceApp(name, frontend, backend, database, designSystem)
	case "blog":
		return generateBlogApp(name, frontend, backend, database, designSystem)
	case "saas":
		return generateSaaSApp(name, frontend, backend, database, designSystem)
	default: // "empty"
		return generateEmptyApp(name, frontend, backend, database, designSystem)
	}
}

func generateAuthApp(name, frontend, backend, database, ds string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app %s is a web application\n\n", name)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n")
	b.WriteString("  avatar is image, optional\n")
	b.WriteString("  created_at is datetime\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page SignIn:\n")
	b.WriteString("  show heading \"Sign In\"\n")
	b.WriteString("  show form with email, password\n")
	b.WriteString("  when form is submitted, call Login\n")
	b.WriteString("  show link \"Forgot password?\" navigates to ForgotPassword\n")
	b.WriteString("  show link \"Create account\" navigates to SignUp\n\n")

	b.WriteString("page SignUp:\n")
	b.WriteString("  show heading \"Create Account\"\n")
	b.WriteString("  show form with name, email, password\n")
	b.WriteString("  when form is submitted, call Register\n\n")

	b.WriteString("page ForgotPassword:\n")
	b.WriteString("  show heading \"Reset Password\"\n")
	b.WriteString("  show form with email\n")
	b.WriteString("  when form is submitted, call RequestReset\n\n")

	b.WriteString("page Profile:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"My Profile\"\n")
	b.WriteString("  fetch current user from GetProfile\n")
	b.WriteString("  show name, email, avatar\n")
	b.WriteString("  show button \"Edit Profile\" navigates to EditProfile\n\n")

	b.WriteString("page EditProfile:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Edit Profile\"\n")
	b.WriteString("  show form with name, email, avatar\n")
	b.WriteString("  when form is submitted, call UpdateProfile\n\n")

	b.WriteString("# ── APIs ──\n\n")
	b.WriteString("api Register:\n")
	b.WriteString("  accepts name, email, password\n")
	b.WriteString("  check email is valid email\n")
	b.WriteString("  check password has minimum 8 characters\n")
	b.WriteString("  create User with name, email, password\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("api Login:\n")
	b.WriteString("  accepts email, password\n")
	b.WriteString("  find User where email matches\n")
	b.WriteString("  check password is correct\n")
	b.WriteString("  respond with token\n\n")

	b.WriteString("api RequestReset:\n")
	b.WriteString("  accepts email\n")
	b.WriteString("  find User where email matches\n")
	b.WriteString("  send password reset email\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("api GetProfile:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  respond with current user\n\n")

	b.WriteString("api UpdateProfile:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  accepts name, email, avatar\n")
	b.WriteString("  update current user with name, email, avatar\n")
	b.WriteString("  respond with success\n\n")

	writeBuildSection(&b, frontend, backend, database, ds)
	return b.String()
}

func generateCRUDApp(name, frontend, backend, database, ds, entityName string, fields []entityField) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app %s is a web application\n\n", name)

	// Singular for model name, ensure capitalized.
	singular := capitalize(strings.TrimSuffix(entityName, "s"))
	plural := entityName
	if !strings.HasSuffix(plural, "s") {
		plural += "s"
	}

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n\n")

	fmt.Fprintf(&b, "data %s:\n", singular)
	for _, f := range fields {
		fmt.Fprintf(&b, "  %s is %s\n", f.Name, f.Type)
	}
	fmt.Fprintf(&b, "  created_by is User\n")
	fmt.Fprintf(&b, "  created_at is datetime\n\n")

	b.WriteString("# ── Pages ──\n\n")
	fmt.Fprintf(&b, "page %sList:\n", singular)
	fmt.Fprintf(&b, "  show heading \"%s\"\n", plural)
	fmt.Fprintf(&b, "  fetch items from List%s\n", plural)
	fmt.Fprintf(&b, "  show a list of items\n")
	fmt.Fprintf(&b, "  each item shows %s\n", fieldNames(fields))
	fmt.Fprintf(&b, "  clicking an item navigates to %sDetail\n", singular)
	fmt.Fprintf(&b, "  show button \"Create New\" navigates to Create%s\n\n", singular)

	fmt.Fprintf(&b, "page %sDetail:\n", singular)
	fmt.Fprintf(&b, "  fetch item from Get%s\n", singular)
	fmt.Fprintf(&b, "  show %s\n", fieldNames(fields))
	fmt.Fprintf(&b, "  show button \"Edit\" navigates to Edit%s\n", singular)
	fmt.Fprintf(&b, "  show button \"Delete\" calls Delete%s\n\n", singular)

	fmt.Fprintf(&b, "page Create%s:\n", singular)
	fmt.Fprintf(&b, "  show heading \"Create %s\"\n", singular)
	fmt.Fprintf(&b, "  show form with %s\n", fieldNames(fields))
	fmt.Fprintf(&b, "  when form is submitted, call Create%s\n\n", singular)

	fmt.Fprintf(&b, "page Edit%s:\n", singular)
	fmt.Fprintf(&b, "  show heading \"Edit %s\"\n", singular)
	fmt.Fprintf(&b, "  fetch item from Get%s\n", singular)
	fmt.Fprintf(&b, "  show form with %s\n", fieldNames(fields))
	fmt.Fprintf(&b, "  when form is submitted, call Update%s\n\n", singular)

	b.WriteString("page Profile:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"My Profile\"\n")
	b.WriteString("  fetch current user from GetProfile\n")
	b.WriteString("  show name, email\n\n")

	b.WriteString("# ── APIs ──\n\n")
	b.WriteString("api SignUp:\n")
	b.WriteString("  accepts name, email, password\n")
	b.WriteString("  create User with name, email, password\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("api Login:\n")
	b.WriteString("  accepts email, password\n")
	b.WriteString("  find User where email matches\n")
	b.WriteString("  check password is correct\n")
	b.WriteString("  respond with token\n\n")

	fmt.Fprintf(&b, "api List%s:\n", plural)
	fmt.Fprintf(&b, "  fetch all %s\n", singular)
	b.WriteString("  respond with items\n\n")

	fmt.Fprintf(&b, "api Get%s:\n", singular)
	b.WriteString("  accepts id\n")
	fmt.Fprintf(&b, "  find %s where id matches\n", singular)
	b.WriteString("  respond with item\n\n")

	fmt.Fprintf(&b, "api Create%s:\n", singular)
	b.WriteString("  requires authentication\n")
	fmt.Fprintf(&b, "  accepts %s\n", fieldNames(fields))
	fmt.Fprintf(&b, "  create %s with %s\n", singular, fieldNames(fields))
	b.WriteString("  respond with success\n\n")

	fmt.Fprintf(&b, "api Update%s:\n", singular)
	b.WriteString("  requires authentication\n")
	fmt.Fprintf(&b, "  accepts id, %s\n", fieldNames(fields))
	fmt.Fprintf(&b, "  update %s with %s\n", singular, fieldNames(fields))
	b.WriteString("  respond with success\n\n")

	fmt.Fprintf(&b, "api Delete%s:\n", singular)
	b.WriteString("  requires authentication\n")
	b.WriteString("  accepts id\n")
	fmt.Fprintf(&b, "  delete %s where id matches\n", singular)
	b.WriteString("  respond with success\n\n")

	b.WriteString("api GetProfile:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  respond with current user\n\n")

	writeBuildSection(&b, frontend, backend, database, ds)
	return b.String()
}

func generateDashboardApp(name, frontend, backend, database, ds string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app %s is a web application\n\n", name)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n")
	b.WriteString("  role is text\n\n")

	b.WriteString("data Metric:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  value is decimal\n")
	b.WriteString("  category is text\n")
	b.WriteString("  recorded_at is datetime\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page Dashboard:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Dashboard\"\n")
	b.WriteString("  fetch stats from GetMetrics\n")
	b.WriteString("  show total users, active sessions, revenue\n")
	b.WriteString("  show chart of metrics over time\n\n")

	b.WriteString("page Users:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"User Management\"\n")
	b.WriteString("  fetch users from ListUsers\n")
	b.WriteString("  show a table of users\n")
	b.WriteString("  each row shows name, email, role\n\n")

	b.WriteString("page Settings:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Settings\"\n")
	b.WriteString("  show form with name, email\n\n")

	b.WriteString("# ── APIs ──\n\n")
	writeStandardAuthAPIs(&b)

	b.WriteString("api GetMetrics:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  fetch all Metric\n")
	b.WriteString("  respond with items\n\n")

	b.WriteString("api ListUsers:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  fetch all User\n")
	b.WriteString("  respond with items\n\n")

	writeBuildSection(&b, frontend, backend, database, ds)
	return b.String()
}

func generateEcommerceApp(name, frontend, backend, database, ds string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app %s is a web application\n\n", name)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n\n")

	b.WriteString("data Product:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  description is text\n")
	b.WriteString("  price is decimal, required\n")
	b.WriteString("  image is image\n")
	b.WriteString("  category is text\n")
	b.WriteString("  stock is number\n\n")

	b.WriteString("data Order:\n")
	b.WriteString("  user is User\n")
	b.WriteString("  total is decimal\n")
	b.WriteString("  status is text\n")
	b.WriteString("  created_at is datetime\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page Home:\n")
	b.WriteString("  show heading \"Welcome to " + name + "\"\n")
	b.WriteString("  fetch products from ListProducts\n")
	b.WriteString("  show a grid of products\n")
	b.WriteString("  each product shows name, price, image\n\n")

	b.WriteString("page ProductDetail:\n")
	b.WriteString("  fetch product from GetProduct\n")
	b.WriteString("  show name, description, price, image\n")
	b.WriteString("  show button \"Add to Cart\" calls AddToCart\n\n")

	b.WriteString("page Cart:\n")
	b.WriteString("  show heading \"Shopping Cart\"\n")
	b.WriteString("  fetch items from GetCart\n")
	b.WriteString("  show a list of items with quantity and price\n")
	b.WriteString("  show total price\n")
	b.WriteString("  show button \"Checkout\" navigates to Checkout\n\n")

	b.WriteString("page Checkout:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Checkout\"\n")
	b.WriteString("  show form with shipping address, payment\n")
	b.WriteString("  when form is submitted, call PlaceOrder\n\n")

	b.WriteString("page Orders:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"My Orders\"\n")
	b.WriteString("  fetch orders from ListOrders\n")
	b.WriteString("  show a list of orders with total, status, date\n\n")

	b.WriteString("# ── APIs ──\n\n")
	writeStandardAuthAPIs(&b)

	b.WriteString("api ListProducts:\n")
	b.WriteString("  fetch all Product\n")
	b.WriteString("  respond with items\n\n")

	b.WriteString("api GetProduct:\n")
	b.WriteString("  accepts id\n")
	b.WriteString("  find Product where id matches\n")
	b.WriteString("  respond with item\n\n")

	b.WriteString("api AddToCart:\n")
	b.WriteString("  accepts product_id, quantity\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("api GetCart:\n")
	b.WriteString("  respond with cart items\n\n")

	b.WriteString("api PlaceOrder:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  accepts shipping_address\n")
	b.WriteString("  create Order with user, total, status\n")
	b.WriteString("  respond with order\n\n")

	b.WriteString("api ListOrders:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  fetch Order where user is current user\n")
	b.WriteString("  respond with items\n\n")

	writeBuildSection(&b, frontend, backend, database, ds)
	return b.String()
}

func generateBlogApp(name, frontend, backend, database, ds string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app %s is a web application\n\n", name)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n")
	b.WriteString("  bio is text\n\n")

	b.WriteString("data Post:\n")
	b.WriteString("  title is text, required\n")
	b.WriteString("  content is text, required\n")
	b.WriteString("  author is User\n")
	b.WriteString("  category is text\n")
	b.WriteString("  published is boolean\n")
	b.WriteString("  created_at is datetime\n\n")

	b.WriteString("data Comment:\n")
	b.WriteString("  content is text, required\n")
	b.WriteString("  author is User\n")
	b.WriteString("  post is Post\n")
	b.WriteString("  created_at is datetime\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page Home:\n")
	b.WriteString("  show heading \"" + name + "\"\n")
	b.WriteString("  fetch posts from ListPosts\n")
	b.WriteString("  show a list of posts\n")
	b.WriteString("  each post shows title, author, date, excerpt\n")
	b.WriteString("  clicking a post navigates to PostDetail\n\n")

	b.WriteString("page PostDetail:\n")
	b.WriteString("  fetch post from GetPost\n")
	b.WriteString("  show title, content, author, date\n")
	b.WriteString("  fetch comments from GetComments\n")
	b.WriteString("  show a list of comments\n")
	b.WriteString("  show form with comment content\n")
	b.WriteString("  when form is submitted, call AddComment\n\n")

	b.WriteString("page Editor:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Write a Post\"\n")
	b.WriteString("  show form with title, content, category\n")
	b.WriteString("  when form is submitted, call CreatePost\n\n")

	b.WriteString("# ── APIs ──\n\n")
	writeStandardAuthAPIs(&b)

	b.WriteString("api ListPosts:\n")
	b.WriteString("  fetch Post where published is true\n")
	b.WriteString("  respond with items\n\n")

	b.WriteString("api GetPost:\n")
	b.WriteString("  accepts id\n")
	b.WriteString("  find Post where id matches\n")
	b.WriteString("  respond with item\n\n")

	b.WriteString("api CreatePost:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  accepts title, content, category\n")
	b.WriteString("  create Post with title, content, category, author\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("api GetComments:\n")
	b.WriteString("  accepts post_id\n")
	b.WriteString("  fetch Comment where post matches post_id\n")
	b.WriteString("  respond with items\n\n")

	b.WriteString("api AddComment:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  accepts post_id, content\n")
	b.WriteString("  create Comment with content, author, post\n")
	b.WriteString("  respond with success\n\n")

	writeBuildSection(&b, frontend, backend, database, ds)
	return b.String()
}

func generateSaaSApp(name, frontend, backend, database, ds string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app %s is a web application\n\n", name)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n")
	b.WriteString("  plan is text\n\n")

	b.WriteString("data Subscription:\n")
	b.WriteString("  user is User\n")
	b.WriteString("  plan is text, required\n")
	b.WriteString("  status is text\n")
	b.WriteString("  started_at is datetime\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page SignIn:\n")
	b.WriteString("  show heading \"Sign In\"\n")
	b.WriteString("  show form with email, password\n")
	b.WriteString("  when form is submitted, call Login\n\n")

	b.WriteString("page SignUp:\n")
	b.WriteString("  show heading \"Get Started\"\n")
	b.WriteString("  show form with name, email, password\n")
	b.WriteString("  when form is submitted, call Register\n\n")

	b.WriteString("page Dashboard:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Dashboard\"\n")
	b.WriteString("  fetch stats from GetDashboard\n")
	b.WriteString("  show total users, active sessions, revenue\n")
	b.WriteString("  show chart of usage over time\n\n")

	b.WriteString("page Settings:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Settings\"\n")
	b.WriteString("  show form with name, email\n")
	b.WriteString("  show current plan and usage\n\n")

	b.WriteString("page Billing:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  show heading \"Billing\"\n")
	b.WriteString("  fetch subscription from GetSubscription\n")
	b.WriteString("  show current plan, status, renewal date\n")
	b.WriteString("  show button \"Upgrade\" navigates to Pricing\n\n")

	b.WriteString("page Pricing:\n")
	b.WriteString("  show heading \"Pricing\"\n")
	b.WriteString("  show pricing tiers: Free, Pro, Enterprise\n\n")

	b.WriteString("# ── APIs ──\n\n")
	writeStandardAuthAPIs(&b)

	b.WriteString("api GetDashboard:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  respond with stats\n\n")

	b.WriteString("api GetSubscription:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  find Subscription where user is current user\n")
	b.WriteString("  respond with item\n\n")

	b.WriteString("api UpgradePlan:\n")
	b.WriteString("  requires authentication\n")
	b.WriteString("  accepts plan\n")
	b.WriteString("  update Subscription with plan\n")
	b.WriteString("  respond with success\n\n")

	writeBuildSection(&b, frontend, backend, database, ds)
	return b.String()
}

func generateEmptyApp(name, frontend, backend, database, ds string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "app %s is a web application\n\n", name)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page Home:\n")
	fmt.Fprintf(&b, "  show heading \"Welcome to %s\"\n\n", name)

	b.WriteString("# ── APIs ──\n\n")
	b.WriteString("api SignUp:\n")
	b.WriteString("  accepts name, email, password\n")
	b.WriteString("  create User with name, email, password\n")
	b.WriteString("  respond with success\n\n")

	writeBuildSection(&b, frontend, backend, database, ds)
	return b.String()
}

// ── Shared Helpers ──

func writeStandardAuthAPIs(b *strings.Builder) {
	b.WriteString("api Register:\n")
	b.WriteString("  accepts name, email, password\n")
	b.WriteString("  check email is valid email\n")
	b.WriteString("  check password has minimum 8 characters\n")
	b.WriteString("  create User with name, email, password\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("api Login:\n")
	b.WriteString("  accepts email, password\n")
	b.WriteString("  find User where email matches\n")
	b.WriteString("  check password is correct\n")
	b.WriteString("  respond with token\n\n")
}

func writeBuildSection(b *strings.Builder, frontend, backend, database, ds string) {
	b.WriteString("# ── Build ──\n\n")
	b.WriteString("build with:\n")
	if !strings.EqualFold(frontend, "None") && frontend != "" {
		if ds != "" {
			fmt.Fprintf(b, "  frontend using %s with %s\n", frontend, ds)
		} else {
			fmt.Fprintf(b, "  frontend using %s\n", frontend)
		}
	}
	fmt.Fprintf(b, "  backend using %s\n", backend)
	fmt.Fprintf(b, "  database using %s\n", database)
	b.WriteString("  deploy to Docker\n")
}

func fieldNames(fields []entityField) string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func summarizeContent(content string) string {
	lines := strings.Split(content, "\n")
	pages := 0
	apis := 0
	models := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ToLower(line))
		if strings.HasPrefix(trimmed, "page ") && strings.HasSuffix(trimmed, ":") {
			pages++
		}
		if strings.HasPrefix(trimmed, "api ") && strings.HasSuffix(trimmed, ":") {
			apis++
		}
		if strings.HasPrefix(trimmed, "data ") && strings.HasSuffix(trimmed, ":") {
			models++
		}
	}
	if pages == 0 && apis == 0 && models == 0 {
		return ""
	}
	return fmt.Sprintf("  %d page(s), %d API(s), %d data model(s), %d lines", pages, apis, models, len(lines))
}

// ── Legacy Functions (kept for compatibility) ──

// loadExampleTemplate reads an example template and replaces the app name.
func loadExampleTemplate(exampleDir, newName string) (string, error) {
	// Try relative to CWD (development).
	path := filepath.Join("examples", exampleDir, "app.human")
	data, err := os.ReadFile(path)
	if err != nil {
		// Try relative to executable.
		exe, _ := os.Executable()
		if exe != "" {
			path = filepath.Join(filepath.Dir(exe), "..", "examples", exampleDir, "app.human")
			data, err = os.ReadFile(path)
		}
		if err != nil {
			return "", err
		}
	}

	content := string(data)
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) >= 2 && strings.HasPrefix(strings.ToLower(lines[0]), "app ") {
		parts := strings.SplitN(lines[0], " ", 3)
		if len(parts) >= 3 {
			content = fmt.Sprintf("app %s %s\n%s", newName, parts[2], lines[1])
		}
	}

	return content, nil
}

func generateHumanMD(name string, appType AppType) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "Project created with Human compiler using the **%s** template.\n\n", appType.Name)
	b.WriteString("## Instructions\n\n")
	b.WriteString("Add project-specific instructions here. These are passed to the LLM\n")
	b.WriteString("when using `/ask`, `/edit`, and `/suggest` commands.\n\n")
	b.WriteString("## Notes\n\n")
	b.WriteString("- Run `/build` to compile this project\n")
	b.WriteString("- Run `/check` to validate syntax\n")
	b.WriteString("- Run `/edit <instruction>` for AI-assisted editing\n")
	return b.String()
}

func generateGitignore() string {
	return `.human/output/
.human/intent/
.human/config.json
*.tmp
node_modules/
.env
`
}

// Prompt asks the user to choose from options with a default value.
func Prompt(scanner *bufio.Scanner, out io.Writer, label string, options []string, defaultVal string) string {
	fmt.Fprintf(out, "%s (%s) [%s]: ", label, strings.Join(options, "/"), defaultVal)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			for _, opt := range options {
				if strings.EqualFold(input, opt) {
					return opt
				}
			}
			return input
		}
	}
	return defaultVal
}

// GenerateTemplate produces a starter app.human file (used by legacy callers).
func GenerateTemplate(name, platform, frontend, backend, database string) string {
	return generateEmptyApp(name, frontend, backend, database, "")
}

// PromptForPorts interactively prompts the user to configure service ports.
func PromptForPorts(in io.Reader, out io.Writer) ir.PortConfig {
	file, ok := in.(*os.File)
	if !ok || file.Fd() != 0 {
		return ir.PortConfig{
			Frontend: 3000,
			Backend:  3001,
			Database: 5432,
		}
	}

	scanner := bufio.NewScanner(in)
	fmt.Fprintf(out, "\nConfigure service ports:\n")

	frontendPort := PromptForPort(scanner, out, "Frontend", 3000)
	backendPort := PromptForPort(scanner, out, "Backend", 3001)
	databasePort := PromptForPort(scanner, out, "Database", 5432)

	return ir.PortConfig{
		Frontend: frontendPort,
		Backend:  backendPort,
		Database: databasePort,
	}
}

// PromptForPort asks the user for a port number with a default value.
func PromptForPort(scanner *bufio.Scanner, out io.Writer, label string, defaultPort int) int {
	for {
		fmt.Fprintf(out, "  %s port [%d]: ", label, defaultPort)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				return defaultPort
			}
			if port, err := strconv.Atoi(input); err == nil && port > 0 && port <= 65535 {
				return port
			}
			fmt.Fprintf(out, "    Invalid port. Please enter a number between 1 and 65535.\n")
			continue
		}
		return defaultPort
	}
}
