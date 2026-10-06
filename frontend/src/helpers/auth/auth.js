// this function check the respond status after fetching and return false for unanothorized
export function checkSessionResponse(resp) {
    if(resp.status === 401 ) {
        return false;
    }
    return true
}