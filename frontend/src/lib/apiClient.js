const BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8000";

export class ApiError extends Error {
  constructor(message, { status, code, details } = {}) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

let authHook = {
  getAccessToken: () => null,
  refreshAccessToken: async () => null,
  onAuthExpired: () => {},
};

export function configureAuthHook(hook) {
  authHook = { ...authHook, ...hook };
}

async function parseResponse(response) {
  const text = await response.text();
  let body = null;

  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      throw new ApiError("The server returned an unreadable response.", {
        status: response.status,
      });
    }
  }

  if (!response.ok || body?.success === false) {
    const errorBody = body?.error || {};
    throw new ApiError(
      errorBody.message || `Request failed (${response.status})`,
      {
        status: response.status,
        code: errorBody.code,
        details: errorBody.details,
      },
    );
  }

  return body?.data ?? null;
}

async function rawRequest(
  path,
  { method = "GET", body, isFormData, token } = {},
) {
  const headers = {};

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  let payload = body;

  if (body && !isFormData) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }

  let response;

  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body: payload,
    });
  } catch {
    // fetch() itself throws (rather than resolving to a response) for
    // network-level failures: the backend isn't running, the URL is
    // wrong, or the request was blocked by CORS. All three look
    // identical to JavaScript, so this can't be more specific than
    // that -- the browser's Network tab is what tells them apart.
    throw new ApiError(
      `Could not reach the backend at ${BASE_URL}. Is it running, and is VITE_API_BASE_URL ` +
        "correct? (Check the browser console/Network tab for a CORS error, which looks the same here.)",
      { status: 0 },
    );
  }

  return parseResponse(response);
}

// request() is what the rest of the app calls. It attaches the
// current access token automatically and, on a single 401, tries one
// silent token refresh before giving up -- so an expired access token
// never surfaces as a confusing error mid-task.
export async function request(path, options = {}) {
  const token = authHook.getAccessToken();

  try {
    return await rawRequest(path, { ...options, token });
  } catch (error) {
    if (error instanceof ApiError && error.status === 401 && token) {
      const refreshed = await authHook.refreshAccessToken();

      if (refreshed) {
        return rawRequest(path, { ...options, token: refreshed });
      }

      authHook.onAuthExpired();
    }

    throw error;
  }
}

export function get(path) {
  return request(path, { method: "GET" });
}

export function post(path, body) {
  return request(path, { method: "POST", body });
}

export function patch(path, body) {
  return request(path, { method: "PATCH", body });
}

export function del(path) {
  return request(path, { method: "DELETE" });
}

export function postForm(path, formData) {
  return request(path, { method: "POST", body: formData, isFormData: true });
}

// Unauthenticated calls (register/login) never attach or refresh a
// token -- rawRequest is used directly so a stale token can't leak in.
export function postPublic(path, body) {
  return rawRequest(path, { method: "POST", body });
}
