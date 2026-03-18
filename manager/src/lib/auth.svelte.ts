import { config, type AuthResponse } from "./state.svelte";


export async function login(username: string, password: string) {
    const response = await fetch(`${config.apiServer}/auth/login`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            username,
            password
        })
    })
    if (response.ok) {
        const authData = await response.json() as AuthResponse;
        config.auth = authData;
        localStorage.setItem("refreshToken", authData.refresh_token);
    } else {
        console.error("Login failed");
    }
}

export async function logout() {
    config.auth = {} as AuthResponse;
    config.refreshToken = '';
    localStorage.removeItem("refreshToken");
}

export async function refreshToken() {
    if (!config.refreshToken) {
        console.log("Cannot refresh access_token, no refresh_token found. Login required.");
        return;
    }
    if (config.auth?.expires_at && new Date(config.auth.expires_at) >  new Date()) {
        console.log("Access token is still valid, skipping refresh.");
        return;
    }

    const response = await fetch(`${config.apiServer}/auth/refresh`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            refresh_token: config.refreshToken
        })
    })
    if (response.ok) {
        const authData = await response.json() as AuthResponse;
        console.log("Refreshed token", authData);
        config.auth = authData;
    } else {
        console.error(`Failed to refresh token`, response);
    }
}

export async function initializeAuth() {
    if (!config.refreshToken) {
        if (import.meta.env.VITE_GORUN_ADMIN_TOKEN) {
            console.log("Admin token fount in environment variables");
            await localStorage.setItem("refreshToken", import.meta.env.VITE_GORUN_ADMIN_TOKEN);
        }

        const token = localStorage.getItem("refreshToken");
        if (token) {
            console.log("Refresh token found in local storage");
            config.refreshToken = token;
            refreshToken();
        } else {
            console.log("No refresh token found. You need to login.");
        }
    }
}

export async function authorizedFetch(url: string, options?: RequestInit): Promise<Response> {
    await refreshToken()

    const headers = new Headers(options?.headers);
    if (config.auth.access_token) {
        headers.set("Authorization", `Bearer ${config.auth.access_token}`);
    }

    return fetch(url, { ...options, headers });
}
