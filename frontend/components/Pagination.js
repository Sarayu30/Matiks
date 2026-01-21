import React from 'react';
import { View, TouchableOpacity, Text, StyleSheet } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';

export default function Pagination({ currentPage, totalPages, onPageChange }) {
  if (totalPages <= 1) return null;

  const renderPageNumbers = () => {
    const pages = [];
    const maxVisible = 5;
    
    let startPage = Math.max(1, currentPage - 2);
    let endPage = Math.min(totalPages, startPage + maxVisible - 1);
    
    if (endPage - startPage + 1 < maxVisible) {
      startPage = Math.max(1, endPage - maxVisible + 1);
    }

    // First page
    if (startPage > 1) {
      pages.push(
        <TouchableOpacity
          key={1}
          style={styles.pageButton}
          onPress={() => onPageChange(1)}
        >
          <Text style={styles.pageText}>1</Text>
        </TouchableOpacity>
      );
      
      if (startPage > 2) {
        pages.push(
          <Text key="ellipsis1" style={styles.ellipsis}>•••</Text>
        );
      }
    }

    // Middle pages
    for (let i = startPage; i <= endPage; i++) {
      pages.push(
        <TouchableOpacity
          key={i}
          style={[styles.pageButton, currentPage === i && styles.activePageButton]}
          onPress={() => onPageChange(i)}
        >
          <Text style={[styles.pageText, currentPage === i && styles.activePageText]}>
            {i}
          </Text>
        </TouchableOpacity>
      );
    }

    // Last page
    if (endPage < totalPages) {
      if (endPage < totalPages - 1) {
        pages.push(
          <Text key="ellipsis2" style={styles.ellipsis}>•••</Text>
        );
      }
      
      pages.push(
        <TouchableOpacity
          key={totalPages}
          style={styles.pageButton}
          onPress={() => onPageChange(totalPages)}
        >
          <Text style={styles.pageText}>{totalPages}</Text>
        </TouchableOpacity>
      );
    }

    return pages;
  };

  return (
    <View style={styles.container}>
      <TouchableOpacity
        style={[styles.navButton, currentPage === 1 && styles.disabledButton]}
        onPress={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
      >
        <MaterialIcons 
          name="chevron-left" 
          size={24} 
          color={currentPage === 1 ? "#BDBDBD" : "#2196F3"} 
        />
        <Text style={[styles.navText, currentPage === 1 && styles.disabledText]}>
          Previous
        </Text>
      </TouchableOpacity>

      <View style={styles.pagesContainer}>
        {renderPageNumbers()}
      </View>

      <TouchableOpacity
        style={[styles.navButton, currentPage === totalPages && styles.disabledButton]}
        onPress={() => onPageChange(currentPage + 1)}
        disabled={currentPage === totalPages}
      >
        <Text style={[styles.navText, currentPage === totalPages && styles.disabledText]}>
          Next
        </Text>
        <MaterialIcons 
          name="chevron-right" 
          size={24} 
          color={currentPage === totalPages ? "#BDBDBD" : "#2196F3"} 
        />
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 15,
    paddingHorizontal: 10,
    backgroundColor: 'white',
    borderTopWidth: 1,
    borderTopColor: '#e0e0e0',
    elevation: 2,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: -1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
  },
  navButton: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 15,
    paddingVertical: 10,
    borderRadius: 8,
    backgroundColor: '#F5F5F5',
    minWidth: 100,
    justifyContent: 'center',
  },
  navText: {
    fontSize: 14,
    fontWeight: '600',
    color: '#2196F3',
    marginHorizontal: 5,
  },
  disabledButton: {
    backgroundColor: '#FAFAFA',
  },
  disabledText: {
    color: '#BDBDBD',
  },
  pagesContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
    justifyContent: 'center',
  },
  pageButton: {
    minWidth: 40,
    height: 40,
    borderRadius: 20,
    justifyContent: 'center',
    alignItems: 'center',
    marginHorizontal: 2,
  },
  activePageButton: {
    backgroundColor: '#2196F3',
    elevation: 3,
    shadowColor: '#2196F3',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.2,
    shadowRadius: 4,
  },
  pageText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#666',
  },
  activePageText: {
    color: 'white',
    fontWeight: 'bold',
  },
  ellipsis: {
    fontSize: 16,
    color: '#999',
    marginHorizontal: 5,
  },
});