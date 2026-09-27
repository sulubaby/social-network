import LoginPage from "@/views/LoginPage.vue";
import PersonalProfile from "@/views/profiles/PersonalProfile.vue";
import GroupsPage from "@/views/group/GroupsPage.vue";
import GroupPage from "@/views/group/GroupPage.vue";
import HomeFeedPage from "@/views/HomeFeedPage.vue";
import ChatsPage from "@/views/ChatsPage.vue";
import NotificationsPage from "@/views/NotificationsPage.vue";
import SearchPage from "@/views/SearchPage.vue";
import SettingsPage from "@/views/SettingsPage.vue";

import { createRouter, createWebHistory } from "vue-router";
import EditProfile from "@/views/profiles/EditProfile.vue";
import UserProfile from "@/views/profiles/UserProfile.vue";

// IDs in links are only references. The API still decides whether the
// signed-in user may view the requested profile or group. These guards keep
// malformed values out of the app before a request is made.
// checks the id is a normal positive number (no letters, no 0, not too big)
function isSafeId(value) {
  const id = Array.isArray(value) ? value[0] : value
  if (typeof id !== "string" || !/^[1-9]\d*$/.test(id)) return false

  return Number.isSafeInteger(Number(id))
}

// bad user id in /user?id= -> go back home
function validateUserProfileRoute(to) {
  return isSafeId(to.query.id) ? true : { path: "/home", replace: true }
}

// bad group id in /groups/:groupId -> go back to the groups list
function validateGroupRoute(to) {
  return isSafeId(to.params.groupId) ? true : { path: "/groups", replace: true }
}

// every page of the app and which component shows it.
// any unknown url goes to /home
const routes = [
  { path: '/', redirect: '/home' },
  { path: '/:pathMatch(.*)*', redirect: '/home' },
  {
    path: "/login",
    component: LoginPage,
  },
  {
    path: "/me",
    component: PersonalProfile,
    alias: "/profile",
  },
  {
    path: "/me/edit",
    component: EditProfile,
  },
  {
    path: "/user",
    component: UserProfile,
    beforeEnter: validateUserProfileRoute,
  },
  {
    path: "/groups",
    component: GroupsPage,
  },
  {
    path: "/groups/:groupId",
    component: GroupPage,
    beforeEnter: validateGroupRoute,
  },

  {
    path: "/home",
    component: HomeFeedPage,
    alias: "/home-feed",
    meta: {
      requiresAuth: true,
    },
  },
  {
    path: "/chats",
    component: ChatsPage,
    meta: {
      requiresAuth: true,
    },
  },
  {
    path: "/notifications",
    component: NotificationsPage,
    meta: {
      requiresAuth: true,
    },
  },
  {
    path: "/search",
    component: SearchPage,
    meta: {
      requiresAuth: true,
    },
  },
  {
    path: "/settings",
    component: SettingsPage,
    meta: {
      requiresAuth: true,
    },
  },
];

// createWebHistory gives normal urls like /home (no # in the url)
export const router = createRouter({
  history: createWebHistory(),
  routes,
});
