// frontend/services/websocket.js
class WebSocketService {
  constructor() {
    this.socket = null;
    this.reconnectInterval = 3000; // 3 seconds
    this.maxReconnectAttempts = 10;
    this.reconnectAttempts = 0;
    this.listeners = {
      score_updates: [],
      leaderboard_snapshot: [],
      initial_data: [],
      connection_change: [],
    };
    this.isConnected = false;
    this.messageQueue = [];
  }

  connect() {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      console.log('WebSocket already connected');
      return;
    }

    const wsUrl = 'ws://localhost:8080/ws';
    console.log('Connecting to WebSocket:', wsUrl);
    
    try {
      this.socket = new WebSocket(wsUrl);

      this.socket.onopen = () => {
        console.log('✅ WebSocket connected successfully');
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.notifyListeners('connection_change', { connected: true });
        
        // Process any queued messages
        this.processMessageQueue();
      };

      this.socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          console.log('📨 WebSocket message received:', data.type);
          this.handleMessage(data);
        } catch (error) {
          console.error('❌ Error parsing WebSocket message:', error, 'Raw:', event.data);
        }
      };

      this.socket.onerror = (error) => {
        console.error('❌ WebSocket error:', error);
        this.isConnected = false;
        this.notifyListeners('connection_change', { connected: false, error: error.message });
      };

      this.socket.onclose = (event) => {
        console.log(`❌ WebSocket disconnected. Code: ${event.code}, Reason: ${event.reason}`);
        this.isConnected = false;
        this.notifyListeners('connection_change', { connected: false });
        
        if (event.code !== 1000) { // Don't reconnect if closed normally
          this.attemptReconnect();
        }
      };
    } catch (error) {
      console.error('❌ Failed to create WebSocket:', error);
      this.attemptReconnect();
    }
  }

  handleMessage(data) {
    const { type, payload } = data;
    
    console.log(`Handling message type: ${type}`, payload);
    
    if (this.listeners[type]) {
      this.notifyListeners(type, payload);
    } else {
      console.warn(`No listeners for message type: ${type}`);
    }
  }

  notifyListeners(type, data) {
    if (!this.listeners[type]) {
      console.warn(`Unknown message type: ${type}`);
      return;
    }

    this.listeners[type].forEach((callback, index) => {
      if (typeof callback === 'function') {
        try {
          callback(data);
        } catch (error) {
          console.error(`Error in listener for ${type}:`, error);
        }
      } else {
        console.warn(`Invalid listener at index ${index} for ${type}`);
      }
    });
  }

  on(type, callback) {
    if (!this.listeners[type]) {
      this.listeners[type] = [];
    }
    
    this.listeners[type].push(callback);
    
    // Return unsubscribe function
    return () => {
      if (this.listeners[type]) {
        this.listeners[type] = this.listeners[type].filter(cb => cb !== callback);
      }
    };
  }

  send(type, data) {
    if (!this.isConnected || !this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.log('Queueing message - WebSocket not connected');
      this.messageQueue.push({ type, data });
      return false;
    }

    try {
      const message = JSON.stringify({ type, data });
      this.socket.send(message);
      return true;
    } catch (error) {
      console.error('Error sending WebSocket message:', error);
      this.messageQueue.push({ type, data });
      return false;
    }
  }

  processMessageQueue() {
    while (this.messageQueue.length > 0 && this.isConnected) {
      const message = this.messageQueue.shift();
      this.send(message.type, message.data);
    }
  }

  attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('Max reconnection attempts reached');
      this.notifyListeners('connection_change', { 
        connected: false, 
        error: 'Max reconnection attempts reached' 
      });
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectInterval * this.reconnectAttempts;
    
    console.log(`Attempting to reconnect in ${delay}ms (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

    setTimeout(() => {
      console.log('Reconnecting now...');
      this.connect();
    }, delay);
  }

  disconnect() {
    console.log('Disconnecting WebSocket...');
    
    if (this.socket) {
      this.socket.close(1000, 'Normal closure');
      this.socket = null;
    }
    
    this.isConnected = false;
    this.reconnectAttempts = 0;
    this.messageQueue = [];
    
    this.notifyListeners('connection_change', { connected: false });
  }

  getConnectionStatus() {
    return {
      connected: this.isConnected,
      readyState: this.socket ? this.socket.readyState : WebSocket.CLOSED,
      reconnectAttempts: this.reconnectAttempts,
    };
  }

  // Helper method to check if WebSocket is supported
  static isSupported() {
    return typeof WebSocket !== 'undefined';
  }

  // Ping method to keep connection alive (optional)
  startPing(interval = 30000) {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
    }
    
    this.pingInterval = setInterval(() => {
      if (this.isConnected) {
        this.send('ping', { timestamp: Date.now() });
      }
    }, interval);
  }

  stopPing() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }
}

// Create singleton instance
export const webSocketService = new WebSocketService();

// Initialize WebSocket if supported
if (WebSocketService.isSupported()) {
  // Auto-connect after a short delay
  setTimeout(() => {
    webSocketService.connect();
  }, 1000);
} else {
  console.warn('WebSocket is not supported in this environment');
}

// Export for debugging
if (typeof window !== 'undefined') {
  window.webSocketService = webSocketService;
}