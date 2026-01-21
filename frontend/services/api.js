const API_BASE_URL = 'http://localhost:8080';

export const getLeaderboard = async (page = 1, limit = 45) => {
    try {
        const response = await fetch(
            `${API_BASE_URL}/leaderboard?page=${page}&limit=${limit}`,
            { cache: 'no-store' }
        );
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error fetching leaderboard:', error);
        return { 
            success: false, 
            users: [], 
            total: 0, 
            page, 
            limit, 
            totalPages: 0 
        };
    }
};

export const searchUsers = async (query, page = 1, limit = 45) => {
    try {
        const response = await fetch(
            `${API_BASE_URL}/search?q=${encodeURIComponent(query)}&page=${page}&limit=${limit}`,
            { cache: 'no-store' }
        );
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error searching:', error);
        return { 
            success: false, 
            users: [], 
            total: 0, 
            page, 
            limit, 
            totalPages: 0 
        };
    }
};

export const getUserRank = async (username) => {
    try {
        const response = await fetch(
            `${API_BASE_URL}/user/rank?username=${encodeURIComponent(username)}`,
            { cache: 'no-store' }
        );
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error getting rank:', error);
        return { success: false, error: 'Failed to get rank' };
    }
};

export const getStats = async () => {
    try {
        const response = await fetch(`${API_BASE_URL}/stats`, { cache: 'no-store' });
        const data = await response.json();
        return data.stats;
    } catch (error) {
        console.error('Error getting stats:', error);
        return null;
    }
};

export const triggerUpdate = async (count = 100) => {
    try {
        const response = await fetch(`${API_BASE_URL}/update?count=${count}`);
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error triggering update:', error);
        return { success: false };
    }
};

export const triggerDemo = async () => {
    try {
        const response = await fetch(`${API_BASE_URL}/demo`);
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error triggering demo:', error);
        return { success: false };
    }
};