import React, { useState } from 'react';
import {
    View,
    Text,
    TextInput,
    FlatList,
    StyleSheet,
    TouchableOpacity,
    SafeAreaView,
    ActivityIndicator,
} from 'react-native';
import { searchUsers, getUserRank } from '../services/api';

const SearchScreen = () => {
    const [query, setQuery] = useState('');
    const [results, setResults] = useState([]);
    const [loading, setLoading] = useState(false);
    const [selectedUser, setSelectedUser] = useState(null);
    const [userDetails, setUserDetails] = useState(null);

    const handleSearch = async () => {
        if (!query.trim()) {
            setResults([]);
            return;
        }

        setLoading(true);
        setSelectedUser(null);
        setUserDetails(null);
        
        const data = await searchUsers(query, 1, 50);
        
        if (data.success) {
            setResults(data.users || []);
        }
        
        setLoading(false);
    };

    const handleUserSelect = async (user) => {
        setSelectedUser(user);
        const rankInfo = await getUserRank(user.username);
        if (rankInfo.success) {
            setUserDetails(rankInfo.data);
        }
    };

    const renderItem = ({ item }) => (
        <TouchableOpacity
            style={styles.resultItem}
            onPress={() => handleUserSelect(item)}>
            <Text style={styles.resultRank}>#{item.rank}</Text>
            <Text style={styles.resultName}>{item.username}</Text>
            <Text style={styles.resultRating}>{item.rating}</Text>
        </TouchableOpacity>
    );

    return (
        <SafeAreaView style={styles.container}>
            <View style={styles.searchBox}>
                <TextInput
                    style={styles.input}
                    placeholder="Search username (prefix)..."
                    value={query}
                    onChangeText={setQuery}
                    onSubmitEditing={handleSearch}
                    autoCapitalize="none"
                    autoCorrect={false}
                />
                <TouchableOpacity style={styles.searchBtn} onPress={handleSearch}>
                    <Text style={styles.searchBtnText}>Search</Text>
                </TouchableOpacity>
            </View>

            {userDetails && (
                <View style={styles.detailsCard}>
                    <Text style={styles.detailsTitle}>Rank Details</Text>
                    <Text>Username: {selectedUser.username}</Text>
                    <Text>Rating: {selectedUser.rating}</Text>
                    <Text>Global Rank: #{userDetails.user.rank}</Text>
                    <Text>Tied with: {userDetails.tieCount - 1} other user(s)</Text>
                    <Text>Percentile: {userDetails.percentile?.toFixed(2)}%</Text>
                </View>
            )}

            {loading ? (
                <ActivityIndicator style={styles.loader} />
            ) : results.length > 0 ? (
                <FlatList
                    data={results}
                    renderItem={renderItem}
                    keyExtractor={(item) => `${item.id}-${item.rank}`}
                    ListHeaderComponent={() => (
                        <Text style={styles.resultsCount}>
                            Found {results.length} result(s) for "{query}"
                        </Text>
                    )}
                />
            ) : query ? (
                <Text style={styles.noResults}>No users found</Text>
            ) : null}
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: '#fff' },
    searchBox: {
        flexDirection: 'row',
        padding: 16,
        borderBottomWidth: 1,
        borderBottomColor: '#dee2e6',
    },
    input: {
        flex: 1,
        borderWidth: 1,
        borderColor: '#ced4da',
        borderRadius: 6,
        paddingHorizontal: 12,
        paddingVertical: 10,
        marginRight: 10,
    },
    searchBtn: {
        backgroundColor: '#007bff',
        paddingHorizontal: 20,
        justifyContent: 'center',
        borderRadius: 6,
    },
    searchBtnText: { color: '#fff', fontWeight: '600' },
    detailsCard: {
        backgroundColor: '#e7f3ff',
        margin: 16,
        padding: 16,
        borderRadius: 8,
        borderLeftWidth: 4,
        borderLeftColor: '#007bff',
    },
    detailsTitle: {
        fontSize: 16,
        fontWeight: 'bold',
        marginBottom: 8,
        color: '#0056b3',
    },
    loader: { marginTop: 40 },
    resultsCount: {
        padding: 16,
        color: '#6c757d',
        fontSize: 14,
        backgroundColor: '#f8f9fa',
    },
    resultItem: {
        flexDirection: 'row',
        padding: 16,
        borderBottomWidth: 1,
        borderBottomColor: '#e9ecef',
        alignItems: 'center',
    },
    resultRank: {
        fontWeight: 'bold',
        marginRight: 12,
        color: '#007bff',
        width: 60,
    },
    resultName: { flex: 1 },
    resultRating: { fontWeight: '600', color: '#28a745' },
    noResults: {
        textAlign: 'center',
        marginTop: 40,
        color: '#6c757d',
        fontSize: 16,
    },
});

export default SearchScreen;