import { viewedProfile as profileData } from "@/data/usersData";
import { checkSessionResponse } from "@/helpers/auth/auth";
import { router } from "@/router/router";

export async function getProfileData(id, count) {
    const resp = await fetch(`/api/profile?id=${id}`, {
        method: "GET",
        credentials: 'include',
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }

    const result = await resp.json().catch(() => null);
    if (!result) {
        throw new Error('Could not load profile')
    }

    if (!resp.ok) {
        if (result.message == "cannot view your own profile this way") {
            // Replace the invalid self-profile entry. Using push here leaves
            // the profile page in browser history, so Back appears to do
            // nothing and returns to the same profile again.
            router.replace("/me")
            return;
        }
        throw new Error(result.message || 'Could not load profile')
    }

    profileData.userInfo.id = Number(id)
    profileData.userInfo.firstName = result.data.UserInfo.FirstName || ''
    profileData.userInfo.lastName = result.data.UserInfo.LastName || ''
    profileData.userInfo.userName = ''
    profileData.userInfo.email = ''
    profileData.userInfo.dob = ''
    profileData.userInfo.avatar = result.data.UserInfo.Avatar || ''
    profileData.userInfo.isPrivate = result.showProfile
        ? (Number(result.data.IsPrivate) === 1 ? 1 : 0)
        : 1
    profileData.numOfFollowers = result.data.NumOfFollowers || 0
    profileData.numOfPosts = result.data.NumOfPosts || 0
    profileData.numOfFollowing = result.data.NumOfFollowing || 0
    profileData.about.bio = result.data.About.Bio || ''
    profileData.about.work = ''
    profileData.about.education = ''
    profileData.about.travel = ''
    profileData.about.intrests = ''
    profileData.about.hobbies = ''
    profileData.about.website = ''
    profileData.about.linkedin = ''
    profileData.about.instgram = ''
    profileData.about.twitter = ''
    profileData.followers = {}
    profileData.following = {}

    profileData.show = result.showProfile;
    profileData.isFollowing = result.followStatus;
    profileData.canMessage = Boolean(result.canMessage);
    if (profileData.show) {
        profileData.userInfo.userName = result.data.UserInfo.UserName || ''
        profileData.userInfo.email = result.data.UserInfo.Email
        profileData.userInfo.dob = result.data.UserInfo.DOB

        profileData.about.work = result.data.About.Work
        profileData.about.education = result.data.About.Education
        profileData.about.travel = result.data.About.Travel
        profileData.about.intrests = result.data.About.interests
        profileData.about.hobbies = result.data.About.Hobbies
        profileData.about.website = result.data.About.Website
        profileData.about.linkedin = result.data.About.Linkedin
        profileData.about.instgram = result.data.About.instagram
        profileData.about.twitter = result.data.About.Twitter
        try {
            const followerResult = await getFollowers(id, count, 0)
            const followingResult = await getFollowing(id, count, 0)
            profileData.followers = followerResult?.data || {}
            profileData.following = followingResult?.data || {}
        } catch {
            throw new Error('Could not load profile connections')
        }
    }

    return result
}

export async function requestFollow(id, method) {
    const resp = await fetch(`/api/profile/follow?targetid=${id}`, {
        method: method,
        credentials: 'include'
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }

    if (!resp.ok) {
        throw new Error("network error, could not connet to server")
    }

    const result = await resp.json();
    return result
}

export async function removeFollower(id) {
    const resp = await fetch(`/api/profile/follower?followerid=${id}`, {
        method: "DELETE",
        credentials: 'include'
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }

    const result = await resp.json().catch(() => null);

    if (!resp.ok || !result?.status) {
        throw new Error(result?.message || "could not remove follower")
    }

    return result
}

export async function getFollowers(id, count, offset = 0) {
    const resp = await fetch(`/api/profile/follow?targetid=${id}&count=${count}&offset=${offset}`, {
        method: "GET",
        credentials: 'include'
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }

    if (!resp.ok) {
        throw new Error('could not connect to network')
    }

    const result = await resp.json();
    return result;
}

export async function getFollowing(id, count, offset = 0) {
    const resp = await fetch(`/api/profile/following?targetid=${id}&count=${count}&offset=${offset}`, {
        method: "GET",
        credentials: 'include'
    });

    if (!checkSessionResponse(resp)) {
        router.replace("/login");
        return;
    }

    if (!resp.ok) {
        throw new Error('could not connect to network')
    }

    const result = await resp.json();
    return result;
}

export async function searchFollows(searchValue = "", targetId) {
    const params = new URLSearchParams({ search: searchValue });
    if (targetId) params.set('targetid', targetId);

    const resp = await fetch(`/api/profile/follows/search?${params.toString()}`, {
        method: "GET",
        credentials: 'include'
    });

    if (!resp.ok) {
        throw new Error("could not connect to server")
    }

    const result = await resp.json();
    return result;
}

export async function searchFollowing(searchValue = "", targetId) {
    const params = new URLSearchParams({ search: searchValue });
    if (targetId) params.set('targetid', targetId);
    
    const resp = await fetch(`/api/profile/following/search?${params.toString()}`, {
        method: "GET",
        credentials: 'include'
    });

    if (!resp.ok) {
        throw new Error("could not connect to server")
    }

    const result = await resp.json();
    return result;
}


export function getMyFollowing(count = 100) {
    return getFollowing('', count)
}