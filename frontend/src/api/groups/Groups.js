import { checkSessionResponse } from "@/helpers/auth/auth";
import { router } from "@/router/router";

export async function getGroups() {
  const response = await fetch("/api/groups", {
    method: "GET",
    credentials: "include",
  });

  if (!checkSessionResponse(response)) {
    router.replace("/login");
    return;
  }

  if (!response.ok) {
    throw new Error("Error: Could not get groups");
  }

  const result = await response.json();
  return result;
}

export async function createGroupApi(groupData) {
  const response = await fetch("/api/groups", {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(groupData),
  });

  if (!checkSessionResponse(response)) {
    router.replace("/login");
    return;
  }

  if (!response.ok) {
    throw new Error("Error: Could not create a new group");
  }

  const result = await response.json();
  return result;
}

export async function getGroup(groupID) {
  const response = await fetch(`/api/groups/${groupID}`, {
    method: "GET",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!checkSessionResponse(response)) {
    router.replace("/login");
    return;
  }

  if (!response.ok) {
    throw new Error("Error: Could not get a group");
  }

  const result = await response.json();
  return result;
}

export async function deleteGroupApi(groupID) {
  const response = await fetch(`/api/groups/${groupID}`, {
    method: "DELETE",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!checkSessionResponse(response)) {
    router.replace("/login");
    return;
  }

  if (!response.ok) {
    throw new Error("Error: Could not delete a group");
  }

  const result = await response.json();
  return result;
}

export async function groupJoinRequest(groupID) {
  const response = await fetch(`/api/groups/${groupID}/join-requests`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!checkSessionResponse(response)) {
    router.replace("/login");
    return;
  }

  if (!response.ok) {
    throw new Error("Error: Could not create a group join request");
  }

  const result = await response.json();
  return result;
}

export async function undoJoinGroup(groupID) {
  const response = await fetch(`/api/groups/${groupID}/join-requests`, {
    method: "DELETE",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!checkSessionResponse(response)) {
    router.replace("/login");
    return;
  }

  if (!response.ok) {
    throw new Error("Error: Could not undo a group join request");
  }

  const result = await response.json();
  return result;
}

async function groupContentRequest(url, options = {}) {
  const response = await fetch(url, {
    credentials: "include",
    ...options,
  });

  if (!checkSessionResponse(response)) {
    router.replace("/login");
    return;
  }

  const result = await response.json();
  if (!response.ok) {
    throw new Error(result.message || "Could not load group content");
  }

  return result;
}

// one page of group posts, newest first
export function getGroupPosts(groupID, { limit = 20, offset = 0 } = {}) {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  return groupContentRequest(`/api/groups/${groupID}/posts?${params}`);
}

export function createGroupPost(groupID, formData) {
  return groupContentRequest(`/api/groups/${groupID}/posts`, {
    method: "POST",
    body: formData,
  });
}

export function getGroupPostComments(groupID, postID, { limit = 20, offset = 0 } = {}) {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  return groupContentRequest(`/api/groups/${groupID}/posts/${postID}/comments?${params}`);
}

export function createGroupPostComment(groupID, postID, content) {
  return groupContentRequest(`/api/groups/${groupID}/posts/${postID}/comments`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}

export function deleteGroupPost(groupID, postID) {
  return groupContentRequest(`/api/groups/${groupID}/posts/${postID}`, {
    method: "DELETE",
  });
}

export function deleteGroupPostComment(groupID, postID, commentID) {
  return groupContentRequest(`/api/groups/${groupID}/posts/${postID}/comments/${commentID}`, {
    method: "DELETE",
  });
}

export function getInviteUsers(groupID) {
  return groupContentRequest(`/api/groups/${groupID}/invite-users`);
}

export function inviteUserToGroup(groupID, userID) {
  return groupContentRequest(`/api/groups/${groupID}/invitations`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ userId: userID }),
  });
}

export function undoGroupInvitation(groupID, invitationID) {
  return groupContentRequest(`/api/groups/${groupID}/invitations/${invitationID}`, {
    method: "DELETE",
  });
}

export function getGroupEvents(groupID) {
  return groupContentRequest(`/api/groups/${groupID}/events`);
}

export function createGroupEvent(groupID, event) {
  return groupContentRequest(`/api/groups/${groupID}/events`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(event),
  });
}

export function setEventRSVP(eventID, response) {
  return groupContentRequest(`/api/events/${eventID}/rsvp`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ response }),
  });
}

export function removeEventRSVP(groupID, eventID) {
  return groupContentRequest(`/api/groups/${groupID}/events/${eventID}/rsvp`, {
    method: "DELETE",
  });
}

export function deleteGroupEvent(groupID, eventID) {
  return groupContentRequest(`/api/groups/${groupID}/events/${eventID}`, {
    method: "DELETE",
  });
}
