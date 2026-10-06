export async function searchInvites(search, groupID = "") {
    const resp = await fetch(`/api/groups/invites/search?search=${search}&groupID=${groupID}`, {
        method: "GET",
        credentials: 'include'
    });

    const result = await resp.json();
    if (!resp.ok) {
        throw new Error(result.message || "error: could not get users")
    }

    console.log(result.data)
    return result;
}