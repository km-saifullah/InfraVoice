import { get, post, patch, del, postForm, postPublic } from "./apiClient";

// ---- Auth ----------------------------------------------------------------
export const AuthApi = {
  register: (name, email, password) =>
    postPublic("/api/v1/auth/register", { name, email, password }),
  login: (email, password) =>
    postPublic("/api/v1/auth/login", { email, password }),
  refresh: (refreshToken) =>
    postPublic("/api/v1/auth/refresh", { refresh_token: refreshToken }),
  logout: (refreshToken) =>
    post("/api/v1/auth/logout", { refresh_token: refreshToken }),
  me: () => get("/api/v1/auth/me"),
};

// ---- Overview (dashboard rollup shown right after sign-in) -----------------
export const OverviewApi = {
  get: () => get("/api/v1/overview"),
};

// ---- Projects --------------------------------------------------------------
export const ProjectsApi = {
  create: (name, description) =>
    post("/api/v1/projects", { name, description }),
  list: () => get("/api/v1/projects"),
  get: (projectId) => get(`/api/v1/projects/${projectId}`),
  update: (projectId, patchBody) =>
    patch(`/api/v1/projects/${projectId}`, patchBody),
  remove: (projectId) => del(`/api/v1/projects/${projectId}`),
};

// ---- Commands (text + voice log) -------------------------------------------
export const CommandsApi = {
  create: (projectId, input, source = "text") =>
    post(`/api/v1/projects/${projectId}/commands`, { input, source }),
  createVoice: (projectId, audioBlob, filename) => {
    const form = new FormData();
    form.append("audio", audioBlob, filename);
    return postForm(`/api/v1/projects/${projectId}/commands/voice`, form);
  },
  list: (projectId) => get(`/api/v1/projects/${projectId}/commands`),
  get: (commandId) => get(`/api/v1/commands/${commandId}`),
};

// ---- Speech (transcription only) -------------------------------------------
export const SpeechApi = {
  transcribe: (projectId, audioBlob, filename) => {
    const form = new FormData();
    form.append("audio", audioBlob, filename);
    return postForm(`/api/v1/projects/${projectId}/speech/transcribe`, form);
  },
};

// ---- AI (text/voice -> structured infrastructure spec) ---------------------
export const AiApi = {
  parse: (projectId, command) =>
    post(`/api/v1/projects/${projectId}/infrastructure/parse`, { command }),
  parseVoice: (projectId, audioBlob, filename) => {
    const form = new FormData();
    form.append("audio", audioBlob, filename);
    return postForm(
      `/api/v1/projects/${projectId}/infrastructure/parse/voice`,
      form,
    );
  },
};

// ---- Infrastructure specifications ------------------------------------------
export const InfrastructureApi = {
  create: (projectId, spec) =>
    post(`/api/v1/projects/${projectId}/infrastructure`, spec),
  list: (projectId) => get(`/api/v1/projects/${projectId}/infrastructure`),
  get: (projectId, specId) =>
    get(`/api/v1/projects/${projectId}/infrastructure/${specId}`),
  update: (projectId, specId, spec) =>
    patch(`/api/v1/projects/${projectId}/infrastructure/${specId}`, spec),
  validate: (projectId, specId) =>
    post(
      `/api/v1/projects/${projectId}/infrastructure/${specId}/validate`,
      undefined,
    ),
  remove: (projectId, specId) =>
    del(`/api/v1/projects/${projectId}/infrastructure/${specId}`),
};

// ---- Terraform: generate (preview .tf files, nothing executed) -------------
export const TerraformGenerateApi = {
  generate: (projectId, specificationId, region) =>
    post(`/api/v1/projects/${projectId}/terraform/generate`, {
      specification_id: specificationId,
      region,
    }),
};

// ---- Terraform: runs (validate/plan/apply/destroy execution) ---------------
export const TerraformRunsApi = {
  start: (projectId, input) =>
    post(`/api/v1/projects/${projectId}/terraform/runs`, input),
  list: (projectId) => get(`/api/v1/projects/${projectId}/terraform/runs`),
  get: (projectId, runId) =>
    get(`/api/v1/projects/${projectId}/terraform/runs/${runId}`),
};
