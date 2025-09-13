// Main Index Page - Entry Point
// Location: ./src/pages/index.tsx

import React, { useEffect } from 'react';
import { useRouter } from 'next/router';
import { Dashboard } from './Dashboard';

export default function IndexPage() {
  const router = useRouter();

  // Redirect to dashboard or show landing page based on your needs
  useEffect(() => {
    // For now, we'll show the dashboard directly
    // In a real app, you might have authentication logic here
    console.log('Multi-Tenant Ingestion Pipeline Dashboard loaded');
  }, []);

  return <Dashboard />;
}

// For static export compatibility
export async function getStaticProps() {
  return {
    props: {},
  };
}