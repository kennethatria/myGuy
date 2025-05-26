const API_URL = 'http://localhost:8080/api/v1';

// Function to login and get token
async function login(email, password) {
  const response = await fetch(`${API_URL}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
  });
  
  const data = await response.json();
  if (!response.ok) {
    throw new Error(`Login failed for ${email}: ${data.error}`);
  }
  return data.token;
}

// Function to create a task
async function createTask(token, task) {
  const response = await fetch(`${API_URL}/tasks`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(task)
  });
  
  const data = await response.json();
  if (!response.ok) {
    throw new Error(`Failed to create task: ${data.error}`);
  }
  return data;
}

// Function to apply for a task
async function applyForTask(token, taskId, application) {
  const response = await fetch(`${API_URL}/tasks/${taskId}/apply`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(application)
  });
  
  const data = await response.json();
  if (!response.ok) {
    throw new Error(`Failed to apply for task: ${data.error}`);
  }
  return data;
}

// Function to send a message
async function sendMessage(token, taskId, message) {
  const response = await fetch(`${API_URL}/tasks/${taskId}/messages`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(message)
  });
  
  const data = await response.json();
  if (!response.ok) {
    throw new Error(`Failed to send message: ${data.error}`);
  }
  return data;
}

async function main() {
  try {
    console.log('Starting to create sample gigs...\n');
    
    // Login all users
    console.log('Logging in users...');
    const aliceToken = await login('alice@example.com', 'alice123');
    console.log('✓ Alice logged in');
    
    const bobToken = await login('bob@example.com', 'bob123');
    console.log('✓ Bob logged in');
    
    const charlieToken = await login('charlie@example.com', 'charlie123');
    console.log('✓ Charlie logged in');
    
    const dianaToken = await login('diana@example.com', 'diana123');
    console.log('✓ Diana logged in\n');
    
    // Create gigs for each user
    console.log('Creating gigs...');
    
    // Alice's gigs
    const aliceTask1 = await createTask(aliceToken, {
      title: "Build React Dashboard Component",
      description: "Need a responsive dashboard component built with React and Tailwind CSS. Should include charts, stats cards, and recent activity feed. Must be mobile-friendly and support dark mode.",
      deadline: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 500
    });
    console.log(`✓ Created: "${aliceTask1.title}" by Alice`);
    
    const aliceTask2 = await createTask(aliceToken, {
      title: "Fix Authentication Bug in Node.js API",
      description: "JWT tokens are not refreshing properly. Need someone experienced with Node.js and JWT authentication to debug and fix the issue. The bug occurs after 15 minutes of inactivity.",
      deadline: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 200
    });
    console.log(`✓ Created: "${aliceTask2.title}" by Alice`);
    
    // Bob's gigs
    const bobTask1 = await createTask(bobToken, {
      title: "Design Logo for Tech Startup",
      description: "Looking for a modern, minimalist logo design for a tech startup called 'CloudSync'. Should work well in both light and dark modes. Need vector files and brand guidelines.",
      deadline: new Date(Date.now() + 5 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 300
    });
    console.log(`✓ Created: "${bobTask1.title}" by Bob`);
    
    const bobTask2 = await createTask(bobToken, {
      title: "Create UI/UX for Mobile App",
      description: "Need complete UI/UX design for a fitness tracking mobile app. Includes wireframes, mockups, and design system. Should follow Material Design principles.",
      deadline: new Date(Date.now() + 14 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 1200
    });
    console.log(`✓ Created: "${bobTask2.title}" by Bob`);
    
    // Charlie's gigs
    const charlieTask1 = await createTask(charlieToken, {
      title: "Write Technical Blog Posts",
      description: "Need 5 technical blog posts about cloud computing and DevOps practices. Each post should be 1000-1500 words. Topics include Kubernetes, CI/CD, and cloud security.",
      deadline: new Date(Date.now() + 10 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 400
    });
    console.log(`✓ Created: "${charlieTask1.title}" by Charlie`);
    
    const charlieTask2 = await createTask(charlieToken, {
      title: "Edit and Proofread API Documentation",
      description: "Review and improve API documentation for clarity and completeness. About 50 endpoints to document. Experience with technical writing and REST APIs required.",
      deadline: new Date(Date.now() + 4 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 150
    });
    console.log(`✓ Created: "${charlieTask2.title}" by Charlie`);
    
    // Diana's gigs
    const dianaTask1 = await createTask(dianaToken, {
      title: "Implement Payment Integration",
      description: "Integrate Stripe payment processing into existing e-commerce platform built with Node.js. Must handle subscriptions, one-time payments, and webhooks.",
      deadline: new Date(Date.now() + 8 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 800
    });
    console.log(`✓ Created: "${dianaTask1.title}" by Diana`);
    
    const dianaTask2 = await createTask(dianaToken, {
      title: "Optimize Database Queries",
      description: "PostgreSQL database needs performance optimization. Several queries taking 5+ seconds. Need someone with strong SQL skills and experience with query optimization.",
      deadline: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString(),
      fee: 350
    });
    console.log(`✓ Created: "${dianaTask2.title}" by Diana\n`);
    
    // Create some applications
    console.log('Creating applications...');
    
    // Bob applies to Alice's React Dashboard task
    await applyForTask(bobToken, aliceTask1.id, {
      proposed_fee: 450,
      message: "Hi Alice! I have extensive experience with React and Tailwind. I've designed many dashboards and can create a beautiful, responsive component for you. Check out my portfolio!"
    });
    console.log('✓ Bob applied to Alice\'s React Dashboard gig');
    
    // Diana applies to Alice's React Dashboard task
    await applyForTask(dianaToken, aliceTask1.id, {
      proposed_fee: 500,
      message: "I've built similar dashboards before using React and Chart.js. I can deliver this within 5 days with full documentation and tests."
    });
    console.log('✓ Diana applied to Alice\'s React Dashboard gig');
    
    // Alice applies to Bob's Logo Design task
    await applyForTask(aliceToken, bobTask1.id, {
      proposed_fee: 280,
      message: "I have some design experience and can create a clean, modern logo that works in both themes. I'll provide all the vector files you need."
    });
    console.log('✓ Alice applied to Bob\'s Logo Design gig');
    
    // Charlie applies to Diana's Payment Integration task
    await applyForTask(charlieToken, dianaTask1.id, {
      proposed_fee: 750,
      message: "I've integrated Stripe multiple times before. I can handle both subscriptions and one-time payments with proper error handling and webhook implementation."
    });
    console.log('✓ Charlie applied to Diana\'s Payment Integration gig\n');
    
    // Create some messages between users
    console.log('Creating messages...');
    
    // Messages about Alice's Authentication Bug task
    await sendMessage(dianaToken, aliceTask2.id, {
      content: "Hi Alice, I'm interested in fixing your JWT authentication bug. Can you provide more details about the token refresh logic?",
      recipient_id: 2 // Alice's ID
    });
    console.log('✓ Diana messaged Alice about the Authentication Bug');
    
    // Messages about Bob's UI/UX task
    await sendMessage(aliceToken, bobTask2.id, {
      content: "Hey Bob, your mobile app project sounds interesting! Do you have any specific design preferences or existing brand guidelines?",
      recipient_id: 3 // Bob's ID
    });
    console.log('✓ Alice messaged Bob about the Mobile App UI/UX');
    
    console.log('\n✅ Sample data created successfully!\n');
    console.log('Summary:');
    console.log('- 8 gigs created (2 per user)');
    console.log('- 4 applications submitted');
    console.log('- 2 message conversations started');
    console.log('\nYou can now login and explore the full functionality!');
    
  } catch (error) {
    console.error('Error:', error.message);
  }
}

// Run the script
main();