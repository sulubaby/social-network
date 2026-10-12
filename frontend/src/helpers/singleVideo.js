// current is the video that is playing now, so only one plays at the same time
let current = null;

export function claimPlayback(video) {
    if (current && current !== video) {
        current.pause();
    }

    current = video;
}

export function releasePlayback(video) {
    if (current === video) {
        current = null;
    }
}
