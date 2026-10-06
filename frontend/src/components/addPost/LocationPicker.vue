<script setup>

import { getSuggestedLocations } from '@/api/common/location';
import { getLocationData } from '@/helpers/common/locationHelpers';
import { ref } from 'vue';

defineProps({
    modelValue: {
        type: String,
        default: ''
    }
});

const emit = defineEmits(['update:modelValue']);

const search = ref('');
const locations = ref([]);
const selectedLocation = ref(null);

let searchTimeout;

function handleLocationSearch() {
    clearTimeout(searchTimeout);

    searchTimeout = setTimeout(async () => {
        if (!search.value.trim()) {
            locations.value = [];
            return;
        }

        try {
            const result = await getSuggestedLocations(search.value);
            
            locations.value = getLocationData(result);

        } catch (error) {
            console.error(error);
            locations.value = [];
        }
    }, 1000);
}

function selectLocation(location) {
    selectedLocation.value = location;

    search.value = `${location.city}, ${location.country}`;

    locations.value = [];
}

function addLocation() {
    if (!selectedLocation.value) {
        return;
    }
    console.log(selectedLocation.value)
    emit('update:modelValue',`${selectedLocation.value.country}, ${(selectedLocation.value.city || selectedLocation.value.state)}:${selectedLocation.value.lat}:${selectedLocation.value.lon}`);

    search.value = '';
    locations.value = [];
    selectedLocation.value = null;
}

function removeLocation() {
    emit('update:modelValue', '');
}

</script>

<template>

    <div class="location-picker">

        <label for="post-location">
            Add location
        </label>

        <div class="location-row">

            <div class="location-input">

                <input
                    id="post-location"
                    type="text"
                    placeholder="Search for a location"
                    v-model="search"
                    @input="handleLocationSearch"
                >

                <div
                    v-if="locations.length"
                    class="location-suggestions"
                >

                    <button
                        v-for="location in locations"
                        :key="location.link"
                        type="button"
                        class="location-suggestion"
                        @click="selectLocation(location)"
                    >
                        <span>
                            {{ location.city }}, {{ location.country }}
                        </span>
                    </button>

                </div>

            </div>

            <button
                type="button"
                @click="addLocation"
            >
                Add
            </button>

        </div>

        <div
            v-if="modelValue"
            class="location-chip"
        >
            <span>{{ modelValue.split(':')[0].trim() }}</span>
        
            <button
                type="button"
                @click="removeLocation"
            >
                ×
            </button>
        </div>

    </div>

</template>

<style scoped>

.location-picker {
    display: flex;
    flex-direction: column;
    gap: 7px;
}

label {
    color: var(--font-color-sub);
    font-family: "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 1px;
    text-transform: uppercase;
}

.location-row {
    display: flex;
    gap: 10px;
}

.location-input {
    position: relative;
    flex: 1;
}

.location-row input {
    width: 100%;
    box-sizing: border-box;
    padding: 12px 14px;
    border: 2px solid var(--main-color);
    border-radius: 5px;
    outline: none;
    background: var(--bg-color);
    color: var(--font-color);
    transition: box-shadow 0.15s ease;
}

.location-row input:focus {
    box-shadow: 3px 3px var(--main-color);
}

.location-row > button {
    padding: 0 18px;
    border: 2px solid var(--main-color);
    border-radius: 5px;
    background: var(--input-focus);
    box-shadow: 3px 3px var(--main-color);
    color: white;
    font-family: "JetBrains Mono", monospace;
    font-size: 10px;
    font-weight: 700;
    transition: transform 0.1s ease, box-shadow 0.1s ease;
}

.location-row > button:hover {
    transform: translate(-1px, -1px);
    box-shadow: 4px 4px var(--main-color);
}

.location-row > button:active {
    transform: translate(2px, 2px);
    box-shadow: 1px 1px var(--main-color);
}

.location-suggestions {
    position: absolute;
    top: calc(100% + 5px);
    left: 0;
    right: 0;
    z-index: 10;
    overflow: hidden;
    border: 2px solid var(--main-color);
    border-radius: 5px;
    background: var(--bg-color);
    box-shadow: 3px 3px var(--main-color);
}

.location-suggestion {
    display: block;
    width: 100%;
    padding: 11px 14px;
    border: 0;
    border-bottom: 1px solid var(--main-color);
    background: var(--bg-color);
    color: var(--font-color);
    text-align: left;
    cursor: pointer;
}

.location-suggestion:last-child {
    border-bottom: 0;
}

.location-suggestion:hover {
    background: var(--input-focus);
    color: white;
}

.location-chip {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    width: fit-content;
    margin-top: 4px;
    padding: 7px 10px 7px 14px;
    border: 2px solid var(--main-color);
    border-radius: 20px;
    background: var(--page-background);
    color: var(--font-color);
    font-size: 12px;
    font-weight: 600;
}

.location-chip button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border: 0;
    border-radius: 50%;
    background: var(--main-color);
    color: white;
    font-size: 12px;
    line-height: 1;
    cursor: pointer;
}

</style>