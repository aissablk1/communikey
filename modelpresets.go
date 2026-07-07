package main

// modelpresets.go — catalogue vérifié de providers d'inférence LLM (couche
// "modèles", cf. modelprovider.go). Chaque preset porte le PROTOCOLE (`Kind`)
// que communikey applique automatiquement — c'est le « routing smart » : l'utilisateur
// tape `communikey model add anthropic` et communikey sait que c'est l'API Messages
// native, pas openai-compatible. Zéro client Go par marque : ~la totalité des
// providers réutilisent l'unique adaptateur openai-compatible (§1 « ne pas
// réinventer », §57 « soustraction ») ; seul l'adaptateur natif Anthropic
// (modelclient_anthropic.go) sert les endpoints en format Messages (anthropic, minimax).
//
// PROVENANCE (§2/§29 — jamais de base_url inventée) :
//   • Src == "clawcodex" : base_url / env / modèle par défaut portés VERBATIM depuis
//     la table réelle du dépôt officiel agentforce314/clawcodex
//     (src/providers/openai_compatible_specs.py + __init__.py), vérifiée le 2026-07-08.
//   • Src == "known" : provider mainstream dont l'endpoint openai-compatible est
//     stable et largement documenté ; à confirmer par l'utilisateur (le modèle par
//     défaut, volatile par nature, est surchargeable via `--model` ou par appel).
//
// Le catalogue n'enregistre RIEN tout seul : il alimente `communikey model add
// <provider>` (écrit une entrée dans models.json) et `communikey model presets`
// (liste le catalogue). models.json reste l'unique source de vérité runtime.

// modelPreset décrit un provider prêt à l'emploi.
type modelPreset struct {
	Label   string // libellé lisible (tables, login)
	Kind    string // protocole : "openai-compatible" | "anthropic"
	BaseURL string // base API ; l'adaptateur ajoute "/chat/completions" (openai) ou "/v1/messages" (anthropic)
	Model   string // modèle par défaut ; surchargeable (--model / par appel)
	AuthEnv string // variable d'env conventionnelle de la clé ; "" = serveur local sans clé
	Src     string // "clawcodex" (base_url vérifiée à la source) | "known" (à valider)
}

// modelPresets — catalogue ordonné-par-nom (accès par clé). Les clés sont l'id
// canonique passé à `communikey model add <id>`.
var modelPresets = map[string]modelPreset{
	// ─── Providers explicites/natifs (base_url vérifiée dans clawcodex/__init__.py) ───
	"openai":     {"OpenAI", "openai-compatible", "https://api.openai.com/v1", "gpt-4o", "OPENAI_API_KEY", "clawcodex"},
	"anthropic":  {"Anthropic (Claude)", "anthropic", "https://api.anthropic.com", "claude-sonnet-4-6", "ANTHROPIC_API_KEY", "clawcodex"},
	"gemini":     {"Google Gemini", "openai-compatible", "https://generativelanguage.googleapis.com/v1beta/openai", "gemini-2.5-pro", "GEMINI_API_KEY", "clawcodex"},
	"deepseek":   {"DeepSeek", "openai-compatible", "https://api.deepseek.com", "deepseek-chat", "DEEPSEEK_API_KEY", "clawcodex"},
	"openrouter": {"OpenRouter", "openai-compatible", "https://openrouter.ai/api/v1", "openrouter/auto", "OPENROUTER_API_KEY", "clawcodex"},
	"zai":        {"Z.ai (GLM)", "openai-compatible", "https://api.z.ai/api/coding/paas/v4", "glm-4.6", "ZAI_API_KEY", "clawcodex"},
	"minimax":    {"MiniMax", "anthropic", "https://api.minimaxi.com/anthropic", "MiniMax-M2", "MINIMAX_API_KEY", "clawcodex"},

	// ─── Gateways / clouds d'inférence openai-compatibles (base_url vérifiée clawcodex) ───
	"nvidia-nim":     {"NVIDIA NIM", "openai-compatible", "https://integrate.api.nvidia.com/v1", "deepseek-ai/deepseek-v4-pro", "NVIDIA_API_KEY", "clawcodex"},
	"atlascloud":     {"AtlasCloud", "openai-compatible", "https://api.atlascloud.ai/v1", "deepseek-ai/deepseek-v4-flash", "ATLASCLOUD_API_KEY", "clawcodex"},
	"wanjie-ark":     {"Wanjie Ark", "openai-compatible", "https://maas-openapi.wanjiedata.com/api/v1", "deepseek-reasoner", "WANJIE_ARK_API_KEY", "clawcodex"},
	"volcengine":     {"Volcengine Ark", "openai-compatible", "https://ark.cn-beijing.volces.com/api/coding/v3", "DeepSeek-V4-Pro", "VOLCENGINE_API_KEY", "clawcodex"},
	"xiaomi-mimo":    {"Xiaomi MiMo", "openai-compatible", "https://token-plan-sgp.xiaomimimo.com/v1", "mimo-v2.5-pro", "XIAOMI_MIMO_TOKEN_PLAN_API_KEY", "clawcodex"},
	"novita":         {"Novita AI", "openai-compatible", "https://api.novita.ai/openai/v1", "deepseek/deepseek-v4-pro", "NOVITA_API_KEY", "clawcodex"},
	"fireworks":      {"Fireworks AI", "openai-compatible", "https://api.fireworks.ai/inference/v1", "accounts/fireworks/models/deepseek-v4-pro", "FIREWORKS_API_KEY", "clawcodex"},
	"siliconflow":    {"SiliconFlow", "openai-compatible", "https://api.siliconflow.com/v1", "deepseek-ai/DeepSeek-V4-Pro", "SILICONFLOW_API_KEY", "clawcodex"},
	"siliconflow-cn": {"SiliconFlow (Chine)", "openai-compatible", "https://api.siliconflow.cn/v1", "deepseek-ai/DeepSeek-V4-Pro", "SILICONFLOW_API_KEY", "clawcodex"},
	"arcee":          {"Arcee AI", "openai-compatible", "https://api.arcee.ai/api/v1", "trinity-large-thinking", "ARCEE_API_KEY", "clawcodex"},
	"moonshot":       {"Moonshot (Kimi)", "openai-compatible", "https://api.moonshot.ai/v1", "kimi-k2.7-code", "MOONSHOT_API_KEY", "clawcodex"},
	"huggingface":    {"HuggingFace Router", "openai-compatible", "https://router.huggingface.co/v1", "deepseek-ai/DeepSeek-V4-Pro", "HUGGINGFACE_API_KEY", "clawcodex"},
	"together":       {"Together AI", "openai-compatible", "https://api.together.xyz/v1", "deepseek-ai/DeepSeek-V4-Pro", "TOGETHER_API_KEY", "clawcodex"},
	"stepfun":        {"StepFun", "openai-compatible", "https://api.stepfun.ai/v1", "step-3.7-flash", "STEPFUN_API_KEY", "clawcodex"},
	"deepinfra":      {"DeepInfra", "openai-compatible", "https://api.deepinfra.com/v1/openai", "deepseek-ai/DeepSeek-V4-Pro", "DEEPINFRA_API_KEY", "clawcodex"},

	// ─── Serveurs LOCAUX openai-compatibles (sans clé — base_url vérifiée clawcodex) ───
	"ollama":   {"Ollama (local)", "openai-compatible", "http://localhost:11434/v1", "deepseek-coder:1.3b", "", "clawcodex"},
	"vllm":     {"vLLM (local)", "openai-compatible", "http://localhost:8000/v1", "deepseek-ai/DeepSeek-V4-Pro", "", "clawcodex"},
	"sglang":   {"SGLang (local)", "openai-compatible", "http://localhost:30000/v1", "deepseek-ai/DeepSeek-V4-Pro", "", "clawcodex"},
	"lmstudio": {"LM Studio (local)", "openai-compatible", "http://localhost:1234/v1", "local-model", "", "known"},
	"jan":      {"Jan (local)", "openai-compatible", "http://localhost:1337/v1", "local-model", "", "known"},
	"localai":  {"LocalAI (local)", "openai-compatible", "http://localhost:8080/v1", "local-model", "", "known"},
	"llamacpp": {"llama.cpp server (local)", "openai-compatible", "http://localhost:8080/v1", "local-model", "", "known"},

	// ─── Mainstream openai-compatibles ajoutés (endpoints stables — Src "known", à valider §29) ───
	"groq":          {"Groq", "openai-compatible", "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", "GROQ_API_KEY", "known"},
	"mistral":       {"Mistral AI", "openai-compatible", "https://api.mistral.ai/v1", "mistral-large-latest", "MISTRAL_API_KEY", "known"},
	"perplexity":    {"Perplexity", "openai-compatible", "https://api.perplexity.ai", "sonar", "PERPLEXITY_API_KEY", "known"},
	"xai":           {"xAI (Grok)", "openai-compatible", "https://api.x.ai/v1", "grok-4", "XAI_API_KEY", "known"},
	"cerebras":      {"Cerebras", "openai-compatible", "https://api.cerebras.ai/v1", "llama-3.3-70b", "CEREBRAS_API_KEY", "known"},
	"sambanova":     {"SambaNova", "openai-compatible", "https://api.sambanova.ai/v1", "Meta-Llama-3.3-70B-Instruct", "SAMBANOVA_API_KEY", "known"},
	"hyperbolic":    {"Hyperbolic", "openai-compatible", "https://api.hyperbolic.xyz/v1", "deepseek-ai/DeepSeek-V3", "HYPERBOLIC_API_KEY", "known"},
	"nebius":        {"Nebius AI Studio", "openai-compatible", "https://api.studio.nebius.com/v1", "deepseek-ai/DeepSeek-V3", "NEBIUS_API_KEY", "known"},
	"baseten":       {"Baseten", "openai-compatible", "https://inference.baseten.co/v1", "deepseek-ai/DeepSeek-V3-0324", "BASETEN_API_KEY", "known"},
	"cohere":        {"Cohere", "openai-compatible", "https://api.cohere.ai/compatibility/v1", "command-r-plus", "COHERE_API_KEY", "known"},
	"lambda":        {"Lambda Inference", "openai-compatible", "https://api.lambda.ai/v1", "deepseek-v3-0324", "LAMBDA_API_KEY", "known"},
	"featherless":   {"Featherless AI", "openai-compatible", "https://api.featherless.ai/v1", "deepseek-ai/DeepSeek-V3", "FEATHERLESS_API_KEY", "known"},
	"upstage":       {"Upstage (Solar)", "openai-compatible", "https://api.upstage.ai/v1", "solar-pro2", "UPSTAGE_API_KEY", "known"},
	"kluster":       {"Kluster.ai", "openai-compatible", "https://api.kluster.ai/v1", "deepseek-ai/DeepSeek-V3", "KLUSTER_API_KEY", "known"},
	"github-models": {"GitHub Models", "openai-compatible", "https://models.github.ai/inference", "openai/gpt-4o", "GITHUB_TOKEN", "known"},
	"vercel":        {"Vercel AI Gateway", "openai-compatible", "https://ai-gateway.vercel.sh/v1", "openai/gpt-4o", "AI_GATEWAY_API_KEY", "known"},
}

// findPreset renvoie le preset et true si l'id existe dans le catalogue.
func findPreset(id string) (modelPreset, bool) {
	p, ok := modelPresets[id]
	return p, ok
}

// presetAuthField construit le champ "auth" de models.json pour un preset :
// "env:VAR" si une clé est attendue, "" pour un serveur local sans clé.
func presetAuthField(p modelPreset) string {
	if p.AuthEnv == "" {
		return ""
	}
	return "env:" + p.AuthEnv
}

// presetToSpec projette un preset du catalogue en modelSpec prête pour models.json.
// C'est le cœur du routing "smart" : le `kind` (protocole) vient du catalogue, pas
// de l'utilisateur. modelOverride/authOverride ("" = aucun) permettent de surcharger
// le modèle par défaut et la source du secret (ex. "vault:anthropic"). Renvoie
// false si l'id est absent du catalogue.
func presetToSpec(id, modelOverride, authOverride string) (modelSpec, bool) {
	p, ok := findPreset(id)
	if !ok {
		return modelSpec{}, false
	}
	model := p.Model
	if modelOverride != "" {
		model = modelOverride
	}
	auth := presetAuthField(p)
	if authOverride != "" {
		auth = authOverride
	}
	return modelSpec{Name: id, Kind: p.Kind, BaseURL: p.BaseURL, Model: model, Auth: auth}, true
}

// sortedPresetIDs renvoie les ids du catalogue triés (ordre stable pour l'affichage).
func sortedPresetIDs() []string {
	ids := make([]string, 0, len(modelPresets))
	for id := range modelPresets {
		ids = append(ids, id)
	}
	// tri par insertion (petit catalogue, zéro dépendance ; évite d'importer "sort"
	// juste pour ça — cohérent avec le style stdlib-minimal du paquet).
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j-1] > ids[j]; j-- {
			ids[j-1], ids[j] = ids[j], ids[j-1]
		}
	}
	return ids
}
