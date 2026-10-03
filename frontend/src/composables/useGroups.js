import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import {
  getGroups,
  createGroupApi,
  groupJoinRequest,
  undoJoinGroup,
  deleteGroupApi,
} from "@/api/groups/Groups";
import { addNotification } from "@/data/notifications";

export function useGroups() {
  const route = useRoute();
  const activeTab = ref("discover");
  const searchInputValue = ref(typeof route.query.search === "string" ? route.query.search : "");
  const modalStatus = ref(false);

  const AllGroupsdata = ref([]);
  onMounted(async () => {
    try {
      const result = await getGroups();

      AllGroupsdata.value = result.groups;
    } catch (error) {
      console.error(error);
    }
  });

  const filteredGroup = computed(() => {
    let groups = AllGroupsdata.value;

    if (activeTab.value === "discover") {
      groups = groups.filter((group) => !group.isMember);
    }

    if (activeTab.value === "my-groups") {
      groups = groups.filter((group) => group.isMember);
    }

    if (searchInputValue.value.trim()) {
      const search = searchInputValue.value.trim().toLowerCase();

      groups = groups.filter((group) => {
        return group.title.toLowerCase().trim().includes(search);
      });
    }

    return groups;
  });

  function tabChanged(tab) {
    activeTab.value = tab;
  }

  function userSearchInput(input) {
    searchInputValue.value = input;
  }

  async function toggleJoinRequest(id) {
    const groupToModify = AllGroupsdata.value.find((group) => group.id === id);

    if (!groupToModify) {
      return;
    }

    try {
      if (!groupToModify.isRequested) {
        const result = await groupJoinRequest(id);
        if (result?.status) {
          groupToModify.isRequested = true;
          addNotification(`Join request sent to ${groupToModify.title}`);
        }
      } else {
        const result = await undoJoinGroup(id);
        if (result?.status) {
          groupToModify.isRequested = false;
        }
      }
    } catch (error) {
      addNotification(error.message || "Could not update your join request", "error");
    }
  }

  function openModal() {
    modalStatus.value = true;
  }

  function closeModal() {
    modalStatus.value = false;
  }

  async function createGroup(data) {
    try {
      const result = await createGroupApi(data);

      const group = result.group;

      // newest groups are shown first
      AllGroupsdata.value.unshift(group);

      closeModal();
      // my new group is under "My Groups", show that tab so it is visible
      activeTab.value = "my-groups";
      addNotification(`${group.title} was created`);
    } catch (error) {
      addNotification(error.message || "Could not create the group", "error");
    }
  }

  async function deleteGroup(groupID) {
    try {
      const result = await deleteGroupApi(groupID);

      if (result?.status) {
        AllGroupsdata.value = AllGroupsdata.value.filter(
          (group) => group.id !== groupID,
        );
      }
    } catch (error) {
      addNotification(error.message || "Could not delete the group", "error");
    }
  }

  return {
    activeTab,
    searchInputValue,
    modalStatus,
    filteredGroup,
    tabChanged,
    userSearchInput,
    toggleJoinRequest,
    openModal,
    closeModal,
    createGroup,
    deleteGroup,
  };
}
