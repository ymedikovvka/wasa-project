import axios from "axios";

const client = axios.create({
  baseURL: "",
  headers: {
    "Content-Type": "application/json",
  },
});

let currentIdentifier = "";

export function setCurrentIdentifier(identifier) {
  currentIdentifier = identifier || "";
}

client.interceptors.request.use((config) => {
  if (currentIdentifier) {
    config.headers.Authorization = `Bearer ${currentIdentifier}`;
  }
  return config;
});

export async function api(method, url, data) {
  const response = await client.request({ method, url, data });
  return response.data;
}

export default client;
