import { checkSessionResponse } from "@/helpers/auth/auth";
import { router } from "@/router/router";

export async function updateUserInfo(userData) {
    const resp = await fetch("/api/user", {
        method: "PATCH",
        credentials: 'include',
        body: JSON.stringify(userData)
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }

    const result = await resp.json();

    if (!resp.ok) {
        throw new Error(result.message || 'error happened')
    }

    return result;
}

export async function updateAbout(data) {
    const resp = await fetch("/api/profile/about", {
        method: "PATCH",
        credentials: 'include',
        body: JSON.stringify(data)
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }
    if (!resp.ok) {
        throw new Error("could not update user data")
    }

    const result = await resp.json();
    return result;
}
// switch my profile between public and private
export async function updatePrivacy(isPrivate) {
    const resp = await fetch("/api/profile/privacy", {
        method: "PATCH",
        credentials: 'include',
        body: JSON.stringify({ isPrivate })
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }

    const result = await resp.json().catch(() => ({}));

    if (!resp.ok) {
        throw new Error(result.message || 'could not change privacy')
    }

    return result;
}
