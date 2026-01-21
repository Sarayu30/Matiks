import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';

export default function UserCard({ user }) {
  return (
    <View style={styles.container}>
      <View style={[styles.rankBadge, user.rank <= 3 && styles.topRankBadge]}>
        <Text style={styles.rankText}>#{user.rank}</Text>
      </View>
      <View style={styles.userInfo}>
        <Text style={styles.username}>{user.username}</Text>
        <View style={styles.ratingContainer}>
          <MaterialIcons name="star" size={14} color="#FFC107" />
          <Text style={styles.rating}>{user.rating}</Text>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: 'white',
    padding: 15,
    marginVertical: 5,
    marginHorizontal: 15,
    borderRadius: 10,
    elevation: 2,
  },
  rankBadge: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#757575',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 15,
  },
  topRankBadge: {
    backgroundColor: '#4CAF50',
  },
  rankText: {
    color: 'white',
    fontWeight: 'bold',
    fontSize: 16,
  },
  userInfo: {
    flex: 1,
  },
  username: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 5,
  },
  ratingContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  rating: {
    marginLeft: 5,
    color: '#666',
    fontSize: 14,
  },
});