import { checkSessionResponse } from "@/helpers/auth/auth";
import { forgetSession, router } from "@/router/router";
import { disconnectRealtime } from "@/services/realtime";

function buildError(data, status, fallback) {
  const error = new Error(data?.message || fallback)

  error.status = status
  error.attemptsLeft = data?.attemptsLeft
  error.retryAfter = data?.retryAfter

  return error
}

export async function checkRegistration(type, input) {
  const response = await fetch('/api/user/registration', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      type,
      input,
    }),
  })

  const data = await response.json()

  if (!response.ok) {
    throw new Error(data.message || 'Could not check availability')
  }

  return data
}

export async function sendEmailCode(email) {
  const response = await fetch('/api/user/send-email-code', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email }),
  })

  const data = await response.json().catch(() => ({}))

  if (!response.ok) {
    throw buildError(data, response.status, 'Could not send verification code')
  }

  return data
}

export async function verifyEmailCode(email, code) {
  const response = await fetch('/api/user/verify-email-code', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, code }),
  })

  const data = await response.json().catch(() => ({}))

  if (!response.ok) {
    throw buildError(data, response.status, 'Could not verify code')
  }

  return data
}

export async function registerUser(userData) {
    const resp = await fetch("/api/user", {
        method: "POST",
        body: userData
    })

    const result = await resp.json().catch(() => ({}))

    if (!resp.ok) {
        throw buildError(result, resp.status, `Registration failed: ${resp.status}`)
    }

    return result
}

export async function loggingSession(userLogger) {
    const resp = await fetch("/api/session", {
        method: "POST",
        credentials: 'include',
        body: JSON.stringify(userLogger)
    });

    const result = await resp.json()

    if (!resp.ok) {
        throw new Error(result.message || `Logging failed: ${resp.status}`)
    }

    // the new cookie is set, the router has to check again
    forgetSession()
    return result
}

export async function logout() {
    const resp = await fetch("/api/session", {
        method: "DELETE",
        credentials: 'include'
    });
    
    if(!checkSessionResponse(resp)) {
        disconnectRealtime()
        forgetSession()
        router.push("/login");
        return
    }

    if (!resp.ok && resp.status != 401) {
        throw new Error("could not logout")
        
    }

    disconnectRealtime()
    forgetSession()
    router.push("/login")
}

export async function authorizeSession() {
    const resp = await fetch("/api/session", {
        method: "GET",
        credentials: "include"
    });

    const result = await resp.json();

    return result;
}
