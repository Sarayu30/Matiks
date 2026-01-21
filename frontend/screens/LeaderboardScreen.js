import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
    View,
    Text,
    FlatList,
    StyleSheet,
    TouchableOpacity,
    ActivityIndicator,
    SafeAreaView,
    RefreshControl,
} from 'react-native';
import { getLeaderboard, getStats, triggerDemo } from '../services/api';

const LeaderboardScreen = ({ navigation }) => {
    const [users, setUsers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [page, setPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [totalUsers, setTotalUsers] = useState(0);
    const [updateCount, setUpdateCount] = useState(0);
    const [stats, setStats] = useState(null);
    
    // Use ref to track current page for intervals
    const pageRef = useRef(1);

    const loadData = async (pageNum, isRefresh = false) => {
        try {
            if (isRefresh) {
                setRefreshing(true);
            } else if (pageNum !== page) {
                setLoading(true);
            }
            
            console.log(`Loading page ${pageNum}...`);
            
            const data = await getLeaderboard(pageNum, 45);
            
            if (data.success) {
                setUsers(data.users || []);
                setTotalPages(data.totalPages || 1);
                setTotalUsers(data.total || 0);
                setUpdateCount(data.updateCount || 0);
                
                // Only update page state if it's different
                if (pageNum !== page) {
                    setPage(pageNum);
                    pageRef.current = pageNum;
                }
                
                console.log(`Loaded page ${pageNum}, total pages: ${data.totalPages}`);
            }
        } catch (error) {
            console.error('Error loading leaderboard:', error);
        } finally {
            setLoading(false);
            setRefreshing(false);
        }
    };

    const loadStats = async () => {
        const statsData = await getStats();
        if (statsData) setStats(statsData);
    };

    // Initial load
    useEffect(() => {
        loadData(1);
        loadStats();
    }, []);

    // Auto-refresh for current page every 3 seconds
    useEffect(() => {
        const interval = setInterval(() => {
            console.log(`Auto-refreshing page ${pageRef.current}`);
            loadData(pageRef.current, false);
        }, 3000); // Every 3 seconds
        
        return () => clearInterval(interval);
    }, []);

    const onRefresh = useCallback(() => {
        loadData(pageRef.current, true);
        loadStats();
    }, []);

    const handleDemo = async () => {
        const result = await triggerDemo();
        if (result.success) {
            // Refresh after demo update
            setTimeout(() => loadData(pageRef.current, true), 1000);
        }
    };

    const handlePageChange = (newPage) => {
        if (newPage >= 1 && newPage <= totalPages) {
            loadData(newPage);
        }
    };

    const renderItem = ({ item, index }) => {
        const isTie = index > 0 && users[index - 1].rating === item.rating;
        
        return (
            <View style={[styles.row, isTie && styles.tieRow]}>
                <Text style={styles.rank}>#{item.rank}</Text>
                <Text style={styles.username}>{item.username}</Text>
                <Text style={styles.rating}>{item.rating}</Text>
            </View>
        );
    };

    const renderPagination = () => {
        if (totalPages <= 1) return null;
        
        const pages = [];
        const maxVisible = 5;
        
        let startPage = Math.max(1, page - Math.floor(maxVisible / 2));
        let endPage = Math.min(totalPages, startPage + maxVisible - 1);
        
        if (endPage - startPage + 1 < maxVisible) {
            startPage = Math.max(1, endPage - maxVisible + 1);
        }
        
        for (let i = startPage; i <= endPage; i++) {
            pages.push(
                <TouchableOpacity
                    key={i}
                    style={[styles.pageBtn, i === page && styles.activePage]}
                    onPress={() => handlePageChange(i)}>
                    <Text style={[styles.pageText, i === page && styles.activeText]}>
                        {i}
                    </Text>
                </TouchableOpacity>
            );
        }
        
        return (
            <View style={styles.pagination}>
                <TouchableOpacity
                    style={[styles.navBtn, page === 1 && styles.disabled]}
                    onPress={() => handlePageChange(page - 1)}
                    disabled={page === 1}>
                    <Text style={styles.navText}>‹ Prev</Text>
                </TouchableOpacity>
                
                {startPage > 1 && (
                    <>
                        <TouchableOpacity
                            style={styles.pageBtn}
                            onPress={() => handlePageChange(1)}>
                            <Text style={styles.pageText}>1</Text>
                        </TouchableOpacity>
                        {startPage > 2 && <Text style={styles.ellipsis}>...</Text>}
                    </>
                )}
                
                {pages}
                
                {endPage < totalPages && (
                    <>
                        {endPage < totalPages - 1 && <Text style={styles.ellipsis}>...</Text>}
                        <TouchableOpacity
                            style={styles.pageBtn}
                            onPress={() => handlePageChange(totalPages)}>
                            <Text style={styles.pageText}>{totalPages}</Text>
                        </TouchableOpacity>
                    </>
                )}
                
                <TouchableOpacity
                    style={[styles.navBtn, page === totalPages && styles.disabled]}
                    onPress={() => handlePageChange(page + 1)}
                    disabled={page === totalPages}>
                    <Text style={styles.navText}>Next ›</Text>
                </TouchableOpacity>
            </View>
        );
    };

    if (loading && users.length === 0) {
        return (
            <SafeAreaView style={styles.center}>
                <ActivityIndicator size="large" />
                <Text>Loading leaderboard...</Text>
            </SafeAreaView>
        );
    }

    return (
        <SafeAreaView style={styles.container}>
            <View style={styles.header}>
                <View>
                    <Text style={styles.title}>🏆 Live Leaderboard</Text>
                    <Text style={styles.subtitle}>
                        {totalUsers.toLocaleString()} users • {updateCount} updates
                        {stats && ` • A:${stats.usersWithA} Z:${stats.usersWithZ}`}
                    </Text>
                </View>
                <TouchableOpacity style={styles.demoBtn} onPress={handleDemo}>
                    <Text style={styles.demoText}>🎯 Demo</Text>
                </TouchableOpacity>
            </View>

            {renderPagination()}

            <View style={styles.infoBar}>
                <Text style={styles.infoText}>
                    Page {page} of {totalPages} • Auto-updates every 3s
                </Text>
            </View>

            <View style={styles.tableHeader}>
                <Text style={styles.headerCell}>Rank</Text>
                <Text style={styles.headerCell}>Username</Text>
                <Text style={styles.headerCell}>Rating</Text>
            </View>

            <FlatList
                data={users}
                renderItem={renderItem}
                keyExtractor={(item) => `${item.id}-${item.rank}-${item.rating}`}
                refreshControl={
                    <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
                }
                ListEmptyComponent={() => (
                    <View style={styles.empty}>
                        <Text>No users found</Text>
                    </View>
                )}
            />

            <TouchableOpacity
                style={styles.searchBtn}
                onPress={() => navigation.navigate('Search')}>
                <Text style={styles.searchText}>🔍 Search Users</Text>
            </TouchableOpacity>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: '#fff' },
    center: { 
        flex: 1, 
        justifyContent: 'center', 
        alignItems: 'center',
        backgroundColor: '#fff'
    },
    header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: 16,
        backgroundColor: '#f8f9fa',
        borderBottomWidth: 1,
        borderBottomColor: '#dee2e6',
    },
    title: { 
        fontSize: 22, 
        fontWeight: 'bold', 
        color: '#212529' 
    },
    subtitle: { 
        fontSize: 12, 
        color: '#6c757d', 
        marginTop: 2 
    },
    demoBtn: {
        backgroundColor: '#007bff',
        paddingHorizontal: 16,
        paddingVertical: 8,
        borderRadius: 6,
    },
    demoText: { 
        color: '#fff', 
        fontWeight: '600', 
        fontSize: 12 
    },
    infoBar: {
        padding: 8,
        backgroundColor: '#e7f3ff',
        alignItems: 'center',
        borderBottomWidth: 1,
        borderBottomColor: '#cfe2ff',
    },
    infoText: {
        fontSize: 12,
        color: '#084298',
        fontWeight: '500',
    },
    pagination: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 12,
        backgroundColor: '#f8f9fa',
        borderBottomWidth: 1,
        borderBottomColor: '#dee2e6',
    },
    navBtn: {
        paddingHorizontal: 12,
        paddingVertical: 8,
        marginHorizontal: 4,
        borderRadius: 4,
        backgroundColor: '#e9ecef',
    },
    navText: { 
        fontSize: 14, 
        fontWeight: '500',
        color: '#495057'
    },
    pageBtn: {
        paddingHorizontal: 12,
        paddingVertical: 8,
        marginHorizontal: 2,
        borderRadius: 4,
        backgroundColor: '#e9ecef',
    },
    activePage: {
        backgroundColor: '#007bff',
    },
    pageText: {
        fontSize: 14,
        fontWeight: '500',
        color: '#495057',
    },
    activeText: {
        color: '#fff',
        fontWeight: 'bold',
    },
    ellipsis: {
        paddingHorizontal: 8,
        color: '#6c757d',
    },
    disabled: { 
        opacity: 0.3 
    },
    tableHeader: {
        flexDirection: 'row',
        backgroundColor: '#343a40',
        paddingVertical: 12,
        paddingHorizontal: 16,
    },
    headerCell: {
        flex: 1,
        color: '#fff',
        fontWeight: 'bold',
        fontSize: 14,
    },
    row: {
        flexDirection: 'row',
        paddingVertical: 12,
        paddingHorizontal: 16,
        borderBottomWidth: 1,
        borderBottomColor: '#e9ecef',
    },
    tieRow: {
        backgroundColor: '#f8f9fa',
    },
    rank: { 
        flex: 1, 
        fontWeight: 'bold', 
        color: '#212529' 
    },
    username: { 
        flex: 2, 
        color: '#495057' 
    },
    rating: { 
        flex: 1, 
        textAlign: 'right', 
        fontWeight: '600', 
        color: '#28a745' 
    },
    empty: {
        padding: 40,
        alignItems: 'center',
    },
    searchBtn: {
        backgroundColor: '#28a745',
        margin: 16,
        padding: 16,
        borderRadius: 8,
        alignItems: 'center',
    },
    searchText: {
        color: '#fff',
        fontWeight: 'bold',
        fontSize: 16,
    },
});

export default LeaderboardScreen;