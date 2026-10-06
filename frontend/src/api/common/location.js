
// this function fetch the location data from api 
//
// @returns array of objects
export async function getSuggestedLocations(searchValue = "") {
    const resp = await fetch(
        `/api/location/search?q=${encodeURIComponent(searchValue)}`,
        {
            method: "GET"
        }
    );

    if (!resp.ok) {
        throw new Error("could not fetch locations");
    }

    const data = await resp.json();


    return data;
}