
const GOOGLE_MAPS_LINK = (lat, lon) => {
    return `https://www.google.com/maps/search/?api=1&query=${lat},${lon}`;
};

export function getLocationData(locations = []) {
    let arr = [];
    locations.forEach(location => {
        arr.push({
            lat: location.lat,
            lon: location.lon,
            link: GOOGLE_MAPS_LINK(location.lat, location.lon),
            city: location.address.city || location.address.state,
            country: location.address.country
        });
    });

    return arr;
}

export function normalizeProfilePost(raw) {
    return {
        id: raw.id,
        userId: raw.userId,
        firstName: raw.firstName,
        lastName: raw.lastName,
        username: raw.username,
        avatarPath: raw.avatarPath,
        content: raw.content,
        imagePath: raw.imagePath,
        allowComments: raw.allowComments,
        location: raw.location,
        groupId: raw.groupId,
        groupName: raw.GroupName ?? raw.groupName ?? '',
        createdAt: raw.createdAt,
        relationship: raw.relationship,
        visibility: raw.visibility,
        visibilityUser: raw.visibilityUser,
        taggedPeople: raw.taggedPeople ?? [],
        likeCount: raw.likeCount ?? 0,
        dislikeCount: raw.disLikeCount ?? 0,
        commentCount: raw.commentCount ?? 0,
        reactionValue: raw.ReactionValue ?? raw.reactionValue ?? 0
    };
}