# Example Configurations

This directory contains example configurations for different domains. You can use these as templates to track repositories in various categories.

## Available Configurations

### Web Frameworks
Track trending web development frameworks and tools.

**Usage:**
```bash
ai-repo-insights -config examples/web-frameworks
```

**Categories:**
- Frontend frameworks (React, Vue, Angular, Svelte)
- Backend frameworks (Express, FastAPI, Django, Flask)
- Fullstack frameworks (Next.js, Nuxt, Remix)
- Meta-frameworks (Astro, Gatsby)
- Build tools (Vite, Webpack, Rollup)

### DevOps
Track trending DevOps tools and infrastructure projects.

**Usage:**
```bash
ai-repo-insights -config examples/devops
```

**Categories:**
- Container orchestration (Docker, Kubernetes)
- Infrastructure as Code (Terraform, Ansible, Pulumi)
- CI/CD tools (GitHub Actions, Jenkins, GitLab CI)
- Monitoring & Observability (Prometheus, Grafana)
- Cloud platforms (AWS, Azure, GCP)

## Creating Custom Configurations

To create a configuration for a new domain:

1. Create a new directory: `examples/your-domain/`
2. Copy all five required files from an existing example
3. Modify `keywords.json` with domain-specific keywords
4. Update `settings.json` with the `filter_domain` name
5. Adapt `writing-guide.md` to the domain

### Required Files

- `keywords.json` - Define include/exclude keywords and categories
- `settings.json` - Configure analysis parameters and domain name
- `languages.json` - Programming languages to track
- `llm.json` - LLM API configuration
- `writing-guide.md` - Writing rules sent verbatim to the LLM

## Switching Between Configurations

Simply use the `-config` flag to point to different configuration directories:

```bash
# Use AI configuration (default)
ai-repo-insights

# Use web frameworks configuration
ai-repo-insights -config examples/web-frameworks

# Use DevOps configuration
ai-repo-insights -config examples/devops

# Use custom configuration
ai-repo-insights -config path/to/your/config
```
