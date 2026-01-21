import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import LeaderboardScreen from './screens/LeaderboardScreen';
import SearchScreen from './screens/SearchScreen';

const Stack = createNativeStackNavigator();

export default function App() {
  // Optional: Add auto-refresh logic here if you want live updates
  useEffect(() => {
    // You can set up WebSocket or polling here for live updates
    // For now, we'll rely on manual refresh
  }, []);

  return (
    <NavigationContainer>
      <Stack.Navigator 
        initialRouteName="Leaderboard"
        screenOptions={{
          headerStyle: {
            backgroundColor: '#007AFF',
          },
          headerTintColor: '#fff',
          headerTitleStyle: {
            fontWeight: 'bold',
          },
        }}
      >
        <Stack.Screen 
          name="Leaderboard" 
          component={LeaderboardScreen}
          options={{ 
            title: '🏆 Leaderboard',
          }}
        />
        <Stack.Screen 
          name="Search" 
          component={SearchScreen}
          options={{ 
            title: '🔍 Search Users',
          }}
        />
      </Stack.Navigator>
    </NavigationContainer>
  );
}