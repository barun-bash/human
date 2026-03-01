package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/ir"
)

type Generator struct{}

func (g Generator) Generate(app *ir.Application, outputDir string) error {
	dirs := []string{
		filepath.Join(outputDir, "alembic"),
		filepath.Join(outputDir, "alembic", "versions"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	files := map[string]string{
		filepath.Join(outputDir, "requirements.txt"):                  generateRequirements(app),
		filepath.Join(outputDir, "main.py"):                           generateMain(app),
		filepath.Join(outputDir, "models.py"):                         generateModels(app),
		filepath.Join(outputDir, "schemas.py"):                        generateSchemas(app),
		filepath.Join(outputDir, "routes.py"):                         generateRoutes(app),
		filepath.Join(outputDir, "auth.py"):                           generateAuth(app),
		filepath.Join(outputDir, "database.py"):                       generateDatabase(app),
		filepath.Join(outputDir, "alembic.ini"):                       generateAlembicIni(app),
		filepath.Join(outputDir, "alembic", "env.py"):                 generateAlembicEnv(app),
		filepath.Join(outputDir, "alembic", "script.py.mako"):         generateAlembicScriptMako(),
		filepath.Join(outputDir, "alembic", "versions", "initial.py"): generateInitialMigration(app),
	}

	// Add policy files if policies are defined
	if len(app.Policies) > 0 {
		files[filepath.Join(outputDir, "policies.py")] = generatePolicies(app)
		files[filepath.Join(outputDir, "authorize.py")] = generateAuthorize(app)
	}

	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	if strings.Contains(s, " ") {
		words := strings.Fields(s)
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
		return strings.Join(words, "")
	}
	if strings.Contains(s, "-") {
		words := strings.Split(s, "-")
		for i, w := range words {
			if w != "" {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		return strings.Join(words, "")
	}
	if strings.Contains(s, "_") {
		words := strings.Split(s, "_")
		for i, w := range words {
			if w != "" {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		return strings.Join(words, "")
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && s[i-1] != ' ' && s[i-1] != '_' && s[i-1] != '-' {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else if r == ' ' || r == '-' {
			result = append(result, '_')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func httpMethod(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "get"), strings.HasPrefix(lower, "list"), strings.HasPrefix(lower, "search"):
		return "get"
	case strings.HasPrefix(lower, "delete"):
		return "delete"
	case strings.HasPrefix(lower, "update"):
		return "put"
	default:
		return "post"
	}
}

func isLoginEndpoint(name string) bool {
	return strings.ToLower(name) == "login"
}

func isSignUpEndpoint(name string) bool {
	lower := strings.ToLower(name)
	return lower == "signup" || lower == "sign_up" || lower == "signUp"
}

func routePath(name string) string {
	stripped := name
	for _, prefix := range []string{"Get", "Create", "Update", "Delete"} {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			stripped = name[len(prefix):]
			break
		}
	}
	return "/" + strings.ReplaceAll(toSnakeCase(stripped), "_", "-")
}

func pythonType(irType string) string {
	switch strings.ToLower(irType) {
	case "text", "email", "url", "file", "image":
		return "str"
	case "number":
		return "int"
	case "decimal":
		return "float"
	case "boolean":
		return "bool"
	case "date":
		return "datetime.date"
	case "datetime":
		return "datetime.datetime"
	case "json":
		return "dict"
	case "enum":
		return "str"
	default:
		return "str"
	}
}

func sqlAlchemyType(irType string) string {
	switch strings.ToLower(irType) {
	case "text", "email", "url", "file", "image", "enum":
		return "String"
	case "number":
		return "Integer"
	case "decimal":
		return "Float"
	case "boolean":
		return "Boolean"
	case "date":
		return "Date"
	case "datetime":
		return "DateTime"
	case "json":
		return "JSON"
	default:
		return "String"
	}
}

func inferModelFromAction(text string) string {
	// Common words that should not be treated as model names
	skip := map[string]bool{
		"current": true, "given": true, "same": true, "new": true,
		"user's": true, "author's": true, "their": true, "own": true,
	}
	words := strings.Fields(text)
	for i, w := range words {
		lower := strings.ToLower(w)
		if lower == "a" || lower == "an" || lower == "the" {
			if i+1 < len(words) {
				candidate := strings.ToLower(words[i+1])
				if skip[candidate] {
					continue
				}
				return toPascalCase(words[i+1])
			}
		} else if lower == "all" {
			if i+1 < len(words) {
				candidate := strings.ToLower(words[i+1])
				if skip[candidate] {
					continue
				}
				name := words[i+1]
				if strings.HasSuffix(name, "s") {
					name = name[:len(name)-1]
				}
				return toPascalCase(name)
			}
		}
	}
	return ""
}

// generatePolicies produces policies.py with role → permission/restriction mappings.
func generatePolicies(app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(`# Generated by Human compiler — do not edit

from typing import List, Optional

class PolicyRule:
    def __init__(self, action: str, model: str, scope: str = "",
                 limit: Optional[int] = None, period: Optional[str] = None,
                 condition: Optional[str] = None):
        self.action = action
        self.model = model
        self.scope = scope
        self.limit = limit
        self.period = period
        self.condition = condition

class PolicyDefinition:
    def __init__(self, name: str, permissions: List[PolicyRule] = None,
                 restrictions: List[PolicyRule] = None):
        self.name = name
        self.permissions = permissions or []
        self.restrictions = restrictions or []

policies: dict[str, PolicyDefinition] = {
`)

	for _, pol := range app.Policies {
		fmt.Fprintf(&sb, "    '%s': PolicyDefinition(\n", pol.Name)
		fmt.Fprintf(&sb, "        name='%s',\n", pol.Name)

		// Permissions
		sb.WriteString("        permissions=[\n")
		for _, perm := range pol.Permissions {
			r := parsePolicyRuleText(perm.Text)
			fmt.Fprintf(&sb, "            PolicyRule(action='%s', model='%s', scope='%s'", r.action, r.model, r.scope)
			if r.limit > 0 {
				fmt.Fprintf(&sb, ", limit=%d", r.limit)
			}
			if r.period != "" {
				fmt.Fprintf(&sb, ", period='%s'", r.period)
			}
			if r.condition != "" {
				fmt.Fprintf(&sb, ", condition='%s'", r.condition)
			}
			sb.WriteString("),\n")
		}
		sb.WriteString("        ],\n")

		// Restrictions
		sb.WriteString("        restrictions=[\n")
		for _, rest := range pol.Restrictions {
			r := parsePolicyRuleText(rest.Text)
			fmt.Fprintf(&sb, "            PolicyRule(action='%s', model='%s', scope='%s'", r.action, r.model, r.scope)
			if r.limit > 0 {
				fmt.Fprintf(&sb, ", limit=%d", r.limit)
			}
			if r.period != "" {
				fmt.Fprintf(&sb, ", period='%s'", r.period)
			}
			if r.condition != "" {
				fmt.Fprintf(&sb, ", condition='%s'", r.condition)
			}
			sb.WriteString("),\n")
		}
		sb.WriteString("        ],\n")

		sb.WriteString("    ),\n")
	}

	sb.WriteString("}\n")
	return sb.String()
}

// generateAuthorize produces authorize.py with a FastAPI dependency for policy enforcement.
func generateAuthorize(app *ir.Application) string {
	return `# Generated by Human compiler — do not edit

from fastapi import Depends, HTTPException, Request, status
from typing import Any
import auth
from policies import policies

def authorize(action: str, model: str):
    """
    Authorization dependency — checks the user's role against defined policies.

    Usage:
        @router.post('/tasks')
        def create_task(current_user = Depends(auth.get_current_user),
                        _authz = Depends(authorize('create', 'task'))):

    Behavior:
        1. If a restriction matches the action+model -> 403 denied
        2. If a permission matches -> allowed (scope attached to request state)
        3. If no rule matches -> allowed (no policy opinion)
    """
    def dependency(current_user: Any = Depends(auth.get_current_user)):
        role = getattr(current_user, 'role', None)
        if not role:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="No role assigned",
            )

        policy = policies.get(role)
        if not policy:
            # No policy defined for this role — allow by default
            return current_user

        # Check restrictions first (deny takes precedence)
        for r in policy.restrictions:
            if r.action == action and (r.model == model or r.model == '*'):
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail=f"{role} cannot {action} {model}",
                )

        # Check permissions — attach scope for downstream query filtering
        for r in policy.permissions:
            if r.action == action and (r.model == model or r.model == '*'):
                current_user._authz_scope = r.scope
                break

        # No matching rule — allow by default (policy has no opinion)
        return current_user

    return dependency
`
}

// parsedPolicyRule is a lightweight parsed representation for Python codegen.
type parsedPolicyRule struct {
	action    string
	model     string
	scope     string
	limit     int
	period    string
	condition string
}

// parsePolicyRuleText extracts structured info from a policy rule's raw text.
func parsePolicyRuleText(text string) parsedPolicyRule {
	lower := strings.ToLower(strings.TrimSpace(text))
	words := strings.Fields(lower)

	r := parsedPolicyRule{}
	if len(words) == 0 {
		return r
	}

	r.action = words[0]

	// Scope detection
	switch {
	case strings.Contains(lower, "only their own"):
		r.scope = "own"
	case strings.Contains(lower, "any of their own"):
		r.scope = "own"
	case containsWord(words, "any"):
		r.scope = "any"
	case containsWord(words, "all"):
		r.scope = "all"
	}

	// Extract model — look for nouns after scope words or action
	skip := map[string]bool{
		"only": true, "their": true, "own": true, "any": true, "all": true,
		"of": true, "the": true, "a": true, "an": true, "and": true,
		"system": true, "up": true, "to": true, "per": true, "unlimited": true,
		"that": true, "which": true, "where": true, "are": true, "is": true,
	}
	for i := len(words) - 1; i >= 1; i-- {
		w := words[i]
		if skip[w] || w == r.action {
			continue
		}
		r.model = singularize(w)
		break
	}

	return r
}

// singularize performs basic English singularization.
func singularize(word string) string {
	if word == "data" || word == "analytics" {
		return word
	}
	if strings.HasSuffix(word, "ies") && len(word) > 3 {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "ses") || strings.HasSuffix(word, "xes") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") && len(word) > 1 {
		return word[:len(word)-1]
	}
	return word
}

// containsWord checks if a word exists in a word list.
func containsWord(words []string, target string) bool {
	for _, w := range words {
		if w == target {
			return true
		}
	}
	return false
}

func generateRequirements(app *ir.Application) string {
	return `fastapi==0.104.1
uvicorn==0.24.0.post1
sqlalchemy==2.0.23
alembic==1.12.1
pydantic[email]==2.5.2
pydantic-settings==2.1.0
python-jose[cryptography]==3.3.0
passlib[bcrypt]==1.7.4
python-multipart==0.0.6
psycopg2-binary==2.9.9
email-validator==2.1.0
`
}

func generateMain(app *ir.Application) string {
	var sb strings.Builder
	appName := app.Name
	if appName == "" {
		appName = "FastAPI App"
	}
	sb.WriteString(fmt.Sprintf(`from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from routes import router

app = FastAPI(title="%s")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router, prefix="/api")

@app.get("/health")
def health_check():
    return {"status": "ok"}
`, appName))

	if app.ErrorHandlers != nil && len(app.ErrorHandlers) > 0 {
		sb.WriteString(`
@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    return JSONResponse(
        status_code=500,
        content={"message": "Internal server error"},
    )
`)
	}

	sb.WriteString(`
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
`)
	return sb.String()
}

func generateModels(app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(`import uuid
from sqlalchemy import Column, Integer, String, Text, Boolean, Float, DateTime, Date, JSON, ForeignKey, Table
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from database import Base

`)

	// First pass: collect has_many_through relationships to generate association tables
	for _, model := range app.Data {
		for _, rel := range model.Relations {
			if rel.Kind == "has_many_through" && rel.Through != "" {
				// Only emit the association table once (check alphabetical ordering)
				if model.Name < rel.Target {
					throughSnake := toSnakeCase(rel.Through)
					sb.WriteString(fmt.Sprintf("%s = Table(\n", throughSnake))
					sb.WriteString(fmt.Sprintf("    '%s',\n", throughSnake))
					sb.WriteString("    Base.metadata,\n")
					sb.WriteString(fmt.Sprintf("    Column('%s_id', String, ForeignKey('%s.id'), primary_key=True),\n", toSnakeCase(model.Name), toSnakeCase(model.Name)))
					sb.WriteString(fmt.Sprintf("    Column('%s_id', String, ForeignKey('%s.id'), primary_key=True),\n", toSnakeCase(rel.Target), toSnakeCase(rel.Target)))
					sb.WriteString(")\n\n")
				}
			}
		}
	}

	for _, model := range app.Data {
		// Skip join models that we've emitted as association tables
		isJoinModel := false
		for _, other := range app.Data {
			for _, rel := range other.Relations {
				if rel.Kind == "has_many_through" && rel.Through == model.Name {
					isJoinModel = true
					break
				}
			}
			if isJoinModel {
				break
			}
		}
		if isJoinModel {
			continue
		}

		sb.WriteString(fmt.Sprintf("class %s(Base):\n", toPascalCase(model.Name)))
		sb.WriteString(fmt.Sprintf("    __tablename__ = '%s'\n\n", toSnakeCase(model.Name)))
		sb.WriteString("    id = Column(String, primary_key=True, index=True, default=lambda: str(uuid.uuid4()))\n")

		for _, field := range model.Fields {
			nullable := "True"
			if field.Required {
				nullable = "False"
			}
			unique := "False"
			if field.Unique {
				unique = "True"
			}
			index := "False"
			if field.Unique {
				index = "True"
			}

			pyType := sqlAlchemyType(field.Type)
			sb.WriteString(fmt.Sprintf("    %s = Column(%s, nullable=%s, unique=%s, index=%s)\n", toSnakeCase(field.Name), pyType, nullable, unique, index))
		}

		sb.WriteString("    created_at = Column(DateTime(timezone=True), server_default=func.now())\n")
		sb.WriteString("    updated_at = Column(DateTime(timezone=True), onupdate=func.now())\n\n")

		for _, rel := range model.Relations {
			if rel.Kind == "belongs_to" {
				sb.WriteString(fmt.Sprintf("    %s_id = Column(String, ForeignKey('%s.id'))\n", toSnakeCase(rel.Target), toSnakeCase(rel.Target)))
				sb.WriteString(fmt.Sprintf("    %s = relationship('%s', back_populates='%s')\n", toSnakeCase(rel.Target), toPascalCase(rel.Target), toSnakeCase(model.Name)+"s"))
			} else if rel.Kind == "has_many" {
				sb.WriteString(fmt.Sprintf("    %s = relationship('%s', back_populates='%s')\n", toSnakeCase(rel.Target)+"s", toPascalCase(rel.Target), toSnakeCase(model.Name)))
			} else if rel.Kind == "has_many_through" {
				throughSnake := toSnakeCase(rel.Through)
				sb.WriteString(fmt.Sprintf("    %s = relationship('%s', secondary=%s, back_populates='%s')\n",
					toSnakeCase(rel.Target)+"s", toPascalCase(rel.Target), throughSnake, toSnakeCase(model.Name)+"s"))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func generateSchemas(app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(`from pydantic import BaseModel, EmailStr, Field
from typing import Optional, List, Dict, Any
import datetime

`)
	for _, model := range app.Data {
		sb.WriteString(fmt.Sprintf("class %sCreate(BaseModel):\n", toPascalCase(model.Name)))
		if len(model.Fields) == 0 {
			sb.WriteString("    pass\n")
		}
		for _, field := range model.Fields {
			pyType := pythonType(field.Type)
			if field.Type == "email" {
				pyType = "EmailStr"
			}
			if !field.Required {
				sb.WriteString(fmt.Sprintf("    %s: Optional[%s] = None\n", toSnakeCase(field.Name), pyType))
			} else {
				sb.WriteString(fmt.Sprintf("    %s: %s\n", toSnakeCase(field.Name), pyType))
			}
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("class %sResponse(BaseModel):\n", toPascalCase(model.Name)))
		sb.WriteString("    id: str\n")
		for _, field := range model.Fields {
			if field.Encrypted {
				continue
			}
			pyType := pythonType(field.Type)
			if field.Type == "email" {
				pyType = "EmailStr"
			}
			if !field.Required {
				sb.WriteString(fmt.Sprintf("    %s: Optional[%s] = None\n", toSnakeCase(field.Name), pyType))
			} else {
				sb.WriteString(fmt.Sprintf("    %s: %s\n", toSnakeCase(field.Name), pyType))
			}
		}
		sb.WriteString("    created_at: Optional[datetime.datetime] = None\n")
		sb.WriteString("    updated_at: Optional[datetime.datetime] = None\n")
		sb.WriteString("\n    class Config:\n        from_attributes = True\n\n")
	}
	return sb.String()
}

func generateRoutes(app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(`from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.orm import Session
from typing import List, Optional, Any
import uuid
import models, schemas, auth
from database import get_db

router = APIRouter()

`)
	for _, api := range app.APIs {
		method := httpMethod(api.Name)
		path := routePath(api.Name)
		isLogin := isLoginEndpoint(api.Name)
		isSignUp := isSignUpEndpoint(api.Name)

		// Build request schema class BEFORE the decorator
		if len(api.Params) > 0 {
			schemaClass := toPascalCase(api.Name) + "Request"
			sb.WriteString(fmt.Sprintf("class %s(schemas.BaseModel):\n", schemaClass))
			for _, p := range api.Params {
				sb.WriteString(fmt.Sprintf("    %s: Any\n", toSnakeCase(p.Name)))
			}
			sb.WriteString("\n")
		}

		// Decorator
		sb.WriteString(fmt.Sprintf("@router.%s('%s')\n", method, path))

		// Function signature — non-default params first, then Depends() params
		var deps []string
		if len(api.Params) > 0 {
			deps = append(deps, fmt.Sprintf("payload: %sRequest", toPascalCase(api.Name)))
		}
		deps = append(deps, "db: Session = Depends(get_db)")
		if api.Auth {
			deps = append(deps, "current_user: Any = Depends(auth.get_current_user)")
		}

		sb.WriteString(fmt.Sprintf("def %s(%s):\n", toSnakeCase(api.Name), strings.Join(deps, ", ")))

		// Validation
		for _, val := range api.Validation {
			if val.Rule == "not_empty" {
				sb.WriteString(fmt.Sprintf("    if not payload.%s:\n", toSnakeCase(val.Field)))
				sb.WriteString(fmt.Sprintf("        raise HTTPException(status_code=400, detail='%s is required')\n", val.Field))
			} else if val.Rule == "max_length" {
				sb.WriteString(fmt.Sprintf("    if payload.%s and len(payload.%s) > %s:\n", toSnakeCase(val.Field), toSnakeCase(val.Field), val.Value))
				sb.WriteString(fmt.Sprintf("        raise HTTPException(status_code=400, detail='%s must be less than %s characters')\n", val.Field, val.Value))
			}
		}

		// Track state for code generation
		queryModelName := ""
		hasCreate := false
		hasReturn := false

		// Generate code for each step
		for _, step := range api.Steps {
			sb.WriteString(fmt.Sprintf("    # %s\n", step.Text))
			switch step.Type {
			case "create":
				modelName := inferModelFromAction(step.Text)
				if modelName != "" {
					hasCreate = true
					if isSignUp {
						sb.WriteString("    hashed_password = auth.get_password_hash(payload.password)\n")
						sb.WriteString(fmt.Sprintf("    new_item = models.%s(\n", modelName))
						for _, p := range api.Params {
							pSnake := toSnakeCase(p.Name)
							if strings.ToLower(p.Name) == "password" {
								sb.WriteString("        password=hashed_password,\n")
							} else {
								sb.WriteString(fmt.Sprintf("        %s=payload.%s,\n", pSnake, pSnake))
							}
						}
						sb.WriteString("    )\n")
					} else {
						sb.WriteString(fmt.Sprintf("    new_item = models.%s(\n", modelName))
						for _, p := range api.Params {
							pSnake := toSnakeCase(p.Name)
							sb.WriteString(fmt.Sprintf("        %s=payload.%s,\n", pSnake, pSnake))
						}
						if api.Auth {
							sb.WriteString("        user_id=current_user.id,\n")
						}
						sb.WriteString("    )\n")
					}
					sb.WriteString("    db.add(new_item)\n    db.commit()\n    db.refresh(new_item)\n")
				}

			case "query":
				modelName := inferModelFromAction(step.Text)
				if modelName != "" && queryModelName == "" {
					queryModelName = modelName
					lowerText := strings.ToLower(step.Text)
					if strings.Contains(lowerText, " by ") {
						// Extract field name after "by"
						parts := strings.SplitN(lowerText, " by ", 2)
						fieldParts := strings.Fields(parts[1])
						queryField := fieldParts[0]
						// Map <model>_id params to the model's id column
						modelCol := queryField
						paramField := queryField
						if strings.HasSuffix(queryField, "_id") {
							modelCol = "id"
						}
						sb.WriteString(fmt.Sprintf("    item = db.query(models.%s).filter(models.%s.%s == payload.%s).first()\n",
							modelName, modelName, modelCol, paramField))
					} else if strings.Contains(lowerText, "all") || strings.Contains(lowerText, "where") {
						sb.WriteString(fmt.Sprintf("    query = db.query(models.%s)\n", modelName))
						sb.WriteString("    items = query.all()\n")
					} else {
						sb.WriteString(fmt.Sprintf("    item = db.query(models.%s).filter(models.%s.id == payload.%s).first()\n",
							modelName, modelName, findIDParam(api)))
					}
				}

			case "condition":
				lowerText := strings.ToLower(step.Text)
				if strings.Contains(lowerText, "does not exist") || strings.Contains(lowerText, "not found") {
					if isLogin {
						sb.WriteString("    if item is None:\n")
						sb.WriteString("        raise HTTPException(status_code=401, detail='Invalid credentials')\n")
					} else {
						label := queryModelName
						if label == "" {
							label = "Item"
						}
						sb.WriteString("    if item is None:\n")
						sb.WriteString(fmt.Sprintf("        raise HTTPException(status_code=404, detail='%s not found')\n", label))
					}
				} else if isLogin && (strings.Contains(lowerText, "password") || strings.Contains(lowerText, "invalid")) {
					sb.WriteString("    if not auth.verify_password(payload.password, item.password):\n")
					sb.WriteString("        raise HTTPException(status_code=401, detail='Invalid credentials')\n")
				}

			case "update":
				lowerText := strings.ToLower(step.Text)
				if strings.Contains(lowerText, "update") && strings.Contains(lowerText, "with") {
					// Bulk field update from payload
					sb.WriteString("    for key, value in payload.model_dump(exclude_unset=True).items():\n")
					sb.WriteString("        setattr(item, key, value)\n")
					sb.WriteString("    db.commit()\n    db.refresh(item)\n")
				} else if strings.Contains(lowerText, "set ") {
					// set field to value
					parts := strings.SplitN(lowerText, "set ", 2)
					if len(parts) == 2 {
						fieldAndValue := strings.SplitN(parts[1], " to ", 2)
						if len(fieldAndValue) == 2 {
							field := strings.TrimSpace(fieldAndValue[0])
							value := strings.TrimSpace(fieldAndValue[1])
							target := "new_item"
							if !hasCreate {
								target = "item"
							}
							if value == "0" || value == "false" || value == "true" {
								sb.WriteString(fmt.Sprintf("    %s.%s = %s\n", target, field, value))
							} else {
								sb.WriteString(fmt.Sprintf("    %s.%s = '%s'\n", target, field, value))
							}
							sb.WriteString("    db.commit()\n")
						}
					}
				}

			case "delete":
				sb.WriteString("    db.delete(item)\n    db.commit()\n")

			case "respond":
				hasReturn = true
				lowerText := strings.ToLower(step.Text)
				if isLogin && strings.Contains(lowerText, "token") {
					sb.WriteString("    token = auth.create_access_token(data={'sub': str(item.id)})\n")
					sb.WriteString("    return {'data': item, 'token': token}\n")
				} else if isSignUp && strings.Contains(lowerText, "token") {
					sb.WriteString("    token = auth.create_access_token(data={'sub': str(new_item.id)})\n")
					sb.WriteString("    return {'data': new_item, 'token': token}\n")
				} else if strings.Contains(lowerText, "created") {
					sb.WriteString("    return {'data': new_item}\n")
				} else if strings.Contains(lowerText, "updated") {
					sb.WriteString("    return {'data': item}\n")
				} else if strings.Contains(lowerText, "deleted") {
					sb.WriteString("    return {'message': 'Deleted successfully'}\n")
				} else if strings.Contains(lowerText, "pagination") || strings.Contains(lowerText, "posts") || strings.Contains(lowerText, "products") || strings.Contains(lowerText, "items") {
					sb.WriteString("    return {'data': items}\n")
				} else if hasCreate {
					sb.WriteString("    return {'data': new_item}\n")
				} else if queryModelName != "" {
					sb.WriteString("    return {'data': item}\n")
				} else {
					sb.WriteString("    return {'message': 'Success'}\n")
				}
			}
		}
		if !hasReturn && len(api.Steps) == 0 {
			sb.WriteString("    return {'message': 'Not implemented'}\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// findIDParam returns the snake_case name of a likely ID param for the endpoint.
func findIDParam(api *ir.Endpoint) string {
	for _, p := range api.Params {
		lower := strings.ToLower(p.Name)
		if strings.HasSuffix(lower, "_id") || strings.HasSuffix(lower, "id") || lower == "slug" {
			return toSnakeCase(p.Name)
		}
	}
	if len(api.Params) > 0 {
		return toSnakeCase(api.Params[0].Name)
	}
	return "id"
}

func generateAuth(app *ir.Application) string {
	return `from datetime import datetime, timedelta
from typing import Optional
from jose import JWTError, jwt
from passlib.context import CryptContext
from fastapi import Depends, HTTPException, status
from fastapi.security import OAuth2PasswordBearer
import models
from database import get_db
from sqlalchemy.orm import Session
import os

SECRET_KEY = os.environ.get("JWT_SECRET", "supersecretkey")
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 60 * 24 * 7 # 7 days default

pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")
oauth2_scheme = OAuth2PasswordBearer(tokenUrl="api/login")

def verify_password(plain_password, hashed_password):
    return pwd_context.verify(plain_password, hashed_password)

def get_password_hash(password):
    return pwd_context.hash(password)

def create_access_token(data: dict, expires_delta: Optional[timedelta] = None):
    to_encode = data.copy()
    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(minutes=15)
    to_encode.update({"exp": expire})
    encoded_jwt = jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)
    return encoded_jwt

def get_current_user(token: str = Depends(oauth2_scheme), db: Session = Depends(get_db)):
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Could not validate credentials",
        headers={"WWW-Authenticate": "Bearer"},
    )
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        user_id: str = payload.get("sub")
        if user_id is None:
            raise credentials_exception
    except JWTError:
        raise credentials_exception
    
    user = db.query(models.User).filter(models.User.id == user_id).first()
    if user is None:
        raise credentials_exception
    return user
`
}

func generateDatabase(app *ir.Application) string {
	return `from sqlalchemy import create_engine
from sqlalchemy.orm import declarative_base, sessionmaker
import os

SQLALCHEMY_DATABASE_URL = os.environ.get("DATABASE_URL", "postgresql://user:password@localhost/dbname")

engine = create_engine(SQLALCHEMY_DATABASE_URL)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

Base = declarative_base()

def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
`
}

func generateAlembicIni(app *ir.Application) string {
	return `[alembic]
script_location = alembic
prepend_sys_path = .
sqlalchemy.url = postgresql://user:password@localhost/dbname

[post_write_hooks]

[loggers]
keys = root,sqlalchemy,alembic

[handlers]
keys = console

[formatters]
keys = generic

[logger_root]
level = WARN
handlers = console
qualname =

[logger_sqlalchemy]
level = WARN
handlers =
qualname = sqlalchemy.engine

[logger_alembic]
level = INFO
handlers =
qualname = alembic

[handler_console]
class = StreamHandler
args = (sys.stderr,)
level = NOTSET
formatter = generic

[formatter_generic]
format = %(levelname)-5.5s [%(name)s] %(message)s
datefmt = %H:%M:%S
`
}

func generateAlembicEnv(app *ir.Application) string {
	return `import os
from logging.config import fileConfig
from sqlalchemy import engine_from_config
from sqlalchemy import pool
from alembic import context
import models

config = context.config

if config.config_file_name is not None:
    fileConfig(config.config_file_name)

target_metadata = models.Base.metadata

def get_url():
    return os.environ.get("DATABASE_URL", config.get_main_option("sqlalchemy.url"))

def run_migrations_offline() -> None:
    url = get_url()
    context.configure(
        url=url,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
    )
    with context.begin_transaction():
        context.run_migrations()

def run_migrations_online() -> None:
    configuration = config.get_section(config.config_ini_section, {})
    configuration["sqlalchemy.url"] = get_url()
    connectable = engine_from_config(
        configuration,
        prefix="sqlalchemy.",
        poolclass=pool.NullPool,
    )
    with connectable.connect() as connection:
        context.configure(
            connection=connection, target_metadata=target_metadata
        )
        with context.begin_transaction():
            context.run_migrations()

if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
`
}

func generateAlembicScriptMako() string {
	return `"""${message}

Revision ID: ${up_revision}
Revises: ${down_revision | comma,n}
Create Date: ${create_date}

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
${imports if imports else ""}


# revision identifiers, used by Alembic.
revision: str = ${repr(up_revision)}
down_revision: Union[str, None] = ${repr(down_revision)}
branch_labels: Union[str, Sequence[str], None] = ${repr(branch_labels)}
depends_on: Union[str, Sequence[str], None] = ${repr(depends_on)}


def upgrade() -> None:
    ${upgrades if upgrades else "pass"}


def downgrade() -> None:
    ${downgrades if downgrades else "pass"}
`
}

func generateInitialMigration(app *ir.Application) string {
	return `"""initial

Revision ID: 000000000000
Revises: 
Create Date: 2026-01-01 00:00:00.000000

"""
from typing import Sequence, Union
from alembic import op
import sqlalchemy as sa

revision: str = '000000000000'
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None

def upgrade() -> None:
    pass

def downgrade() -> None:
    pass
`
}
