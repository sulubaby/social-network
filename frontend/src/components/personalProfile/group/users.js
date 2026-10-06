export const avatarUrl = (p) => (p ? `/uploads/${p}` : '');

export const fullName = (u) =>
    u ? [u.FirstName, u.LastName].filter(Boolean).join(' ') : '';

export const userKey = (u) =>
    String(
        u.UserID ??
        u.UserName ??
        u.Email ??
        `${u.FirstName}-${u.LastName}`
    );

export const normalizeMember = (u) => ({
    UserID: u.UserID,
    Avatar: u.Avatar,
    FirstName: u.FirstName,
    LastName: u.LastName,
    UserName: u.UserName,
    Email: u.Email
});
