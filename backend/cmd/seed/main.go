package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int
	Username string
	Email    string
	Password string
	FullName string
}

type Task struct {
	ID          int
	Title       string
	Description string
	CreatedBy   int
	AssignedTo  *int
	Status      string
	Deadline    time.Time
	Fee         float64
}

func main() {
	// Load environment variables
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Connected to database successfully!")

	// Create test users
	users := []User{
		{Username: "alice_dev", Email: "alice@example.com", Password: "alice123", FullName: "Alice Johnson"},
		{Username: "bob_designer", Email: "bob@example.com", Password: "bob123", FullName: "Bob Smith"},
		{Username: "charlie_writer", Email: "charlie@example.com", Password: "charlie123", FullName: "Charlie Brown"},
		{Username: "diana_coder", Email: "diana@example.com", Password: "diana123", FullName: "Diana Prince"},
	}

	// Hash passwords and insert users
	for i := range users {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(users[i].Password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}

		var userID int
		err = db.QueryRow(ctx, `
			INSERT INTO users (username, email, password_hash, full_name, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (username) DO UPDATE SET
				email = EXCLUDED.email,
				password_hash = EXCLUDED.password_hash,
				full_name = EXCLUDED.full_name,
				updated_at = NOW()
			RETURNING id
		`, users[i].Username, users[i].Email, string(hashedPassword), users[i].FullName).Scan(&userID)
		
		if err != nil {
			log.Fatalf("Failed to insert user %s: %v", users[i].Username, err)
		}
		users[i].ID = userID
		fmt.Printf("Created/Updated user: %s (ID: %d)\n", users[i].Username, userID)
	}

	// Create tasks (gigs) for each user
	tasks := []Task{
		// Alice's tasks
		{
			Title:       "Build React Dashboard Component",
			Description: "Need a responsive dashboard component built with React and Tailwind CSS. Should include charts, stats cards, and recent activity feed.",
			CreatedBy:   users[0].ID,
			Status:      "open",
			Deadline:    time.Now().Add(7 * 24 * time.Hour),
			Fee:         500,
		},
		{
			Title:       "Fix Authentication Bug in Node.js API",
			Description: "JWT tokens are not refreshing properly. Need someone experienced with Node.js and JWT authentication to debug and fix the issue.",
			CreatedBy:   users[0].ID,
			Status:      "in_progress",
			AssignedTo:  &users[3].ID, // Assigned to Diana
			Deadline:    time.Now().Add(3 * 24 * time.Hour),
			Fee:         200,
		},

		// Bob's tasks
		{
			Title:       "Design Logo for Tech Startup",
			Description: "Looking for a modern, minimalist logo design for a tech startup. Should work well in both light and dark modes.",
			CreatedBy:   users[1].ID,
			Status:      "open",
			Deadline:    time.Now().Add(5 * 24 * time.Hour),
			Fee:         300,
		},
		{
			Title:       "Create UI/UX for Mobile App",
			Description: "Need complete UI/UX design for a fitness tracking mobile app. Includes wireframes, mockups, and design system.",
			CreatedBy:   users[1].ID,
			Status:      "open",
			Deadline:    time.Now().Add(14 * 24 * time.Hour),
			Fee:         1200,
		},

		// Charlie's tasks
		{
			Title:       "Write Technical Blog Posts",
			Description: "Need 5 technical blog posts about cloud computing and DevOps practices. Each post should be 1000-1500 words.",
			CreatedBy:   users[2].ID,
			Status:      "open",
			Deadline:    time.Now().Add(10 * 24 * time.Hour),
			Fee:         400,
		},
		{
			Title:       "Edit and Proofread API Documentation",
			Description: "Review and improve API documentation for clarity and completeness. Experience with technical writing required.",
			CreatedBy:   users[2].ID,
			Status:      "in_progress",
			AssignedTo:  &users[0].ID, // Assigned to Alice
			Deadline:    time.Now().Add(4 * 24 * time.Hour),
			Fee:         150,
		},

		// Diana's tasks
		{
			Title:       "Implement Payment Integration",
			Description: "Integrate Stripe payment processing into existing e-commerce platform. Must handle subscriptions and one-time payments.",
			CreatedBy:   users[3].ID,
			Status:      "open",
			Deadline:    time.Now().Add(8 * 24 * time.Hour),
			Fee:         800,
		},
		{
			Title:       "Optimize Database Queries",
			Description: "PostgreSQL database needs performance optimization. Several queries taking too long. Need someone with strong SQL skills.",
			CreatedBy:   users[3].ID,
			Status:      "completed",
			AssignedTo:  &users[2].ID, // Assigned to Charlie
			Deadline:    time.Now().Add(-2 * 24 * time.Hour), // Past deadline
			Fee:         350,
		},
	}

	// Insert tasks
	for i := range tasks {
		var taskID int
		err = db.QueryRow(ctx, `
			INSERT INTO tasks (title, description, created_by, assigned_to, status, deadline, fee, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			RETURNING id
		`, tasks[i].Title, tasks[i].Description, tasks[i].CreatedBy, tasks[i].AssignedTo, 
		   tasks[i].Status, tasks[i].Deadline, tasks[i].Fee).Scan(&taskID)
		
		if err != nil {
			log.Fatalf("Failed to insert task: %v", err)
		}
		tasks[i].ID = taskID
		fmt.Printf("Created task: %s (ID: %d)\n", tasks[i].Title, taskID)
	}

	// Create applications for open tasks
	applications := []struct {
		TaskID      int
		ApplicantID int
		ProposedFee float64
		Message     string
		Status      string
	}{
		// Bob applies to Alice's React Dashboard task
		{
			TaskID:      tasks[0].ID,
			ApplicantID: users[1].ID,
			ProposedFee: 450,
			Message:     "Hi Alice! I have extensive experience with React and Tailwind. I can create a beautiful, responsive dashboard for you. Check out my portfolio!",
			Status:      "pending",
		},
		// Diana applies to Alice's React Dashboard task
		{
			TaskID:      tasks[0].ID,
			ApplicantID: users[3].ID,
			ProposedFee: 500,
			Message:     "I've built similar dashboards before. I can deliver this within 5 days with full documentation.",
			Status:      "pending",
		},
		// Alice applies to Bob's Logo Design task
		{
			TaskID:      tasks[2].ID,
			ApplicantID: users[0].ID,
			ProposedFee: 280,
			Message:     "I have some design experience and can create a clean, modern logo that works in both themes.",
			Status:      "pending",
		},
		// Charlie applies to Diana's Payment Integration task
		{
			TaskID:      tasks[6].ID,
			ApplicantID: users[2].ID,
			ProposedFee: 750,
			Message:     "I've integrated Stripe multiple times before. I can handle both subscriptions and one-time payments with proper error handling.",
			Status:      "pending",
		},
	}

	// Insert applications
	for _, app := range applications {
		_, err = db.Exec(ctx, `
			INSERT INTO applications (task_id, applicant_id, proposed_fee, message, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, app.TaskID, app.ApplicantID, app.ProposedFee, app.Message, app.Status)
		
		if err != nil {
			log.Printf("Warning: Failed to insert application: %v", err)
		}
	}
	fmt.Println("Created applications")

	// Create messages between users about tasks
	messages := []struct {
		TaskID      int
		SenderID    int
		RecipientID int
		Content     string
	}{
		// Conversation about Authentication Bug task (Alice & Diana)
		{
			TaskID:      tasks[1].ID,
			SenderID:    users[0].ID,
			RecipientID: users[3].ID,
			Content:     "Hi Diana, thanks for taking on this task. The main issue is with the refresh token logic.",
		},
		{
			TaskID:      tasks[1].ID,
			SenderID:    users[3].ID,
			RecipientID: users[0].ID,
			Content:     "No problem! I'll start by reviewing the token middleware. Do you have any error logs?",
		},
		{
			TaskID:      tasks[1].ID,
			SenderID:    users[0].ID,
			RecipientID: users[3].ID,
			Content:     "Yes, I'll send them over. The error happens after about 15 minutes of inactivity.",
		},

		// Conversation about API Documentation task (Charlie & Alice)
		{
			TaskID:      tasks[5].ID,
			SenderID:    users[2].ID,
			RecipientID: users[0].ID,
			Content:     "Hi Alice, I've assigned you the documentation task. Let me know if you need any clarification.",
		},
		{
			TaskID:      tasks[5].ID,
			SenderID:    users[0].ID,
			RecipientID: users[2].ID,
			Content:     "Thanks Charlie! I'll review the current docs and create a plan for improvements.",
		},

		// Conversation about Database Optimization (completed task)
		{
			TaskID:      tasks[7].ID,
			SenderID:    users[3].ID,
			RecipientID: users[2].ID,
			Content:     "Charlie, thanks for optimizing those queries! The performance improvement is amazing.",
		},
		{
			TaskID:      tasks[7].ID,
			SenderID:    users[2].ID,
			RecipientID: users[3].ID,
			Content:     "Glad I could help! The main issue was missing indexes. Everything should run much faster now.",
		},
	}

	// Insert messages
	for _, msg := range messages {
		_, err = db.Exec(ctx, `
			INSERT INTO messages (task_id, sender_id, recipient_id, content, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, msg.TaskID, msg.SenderID, msg.RecipientID, msg.Content)
		
		if err != nil {
			log.Printf("Warning: Failed to insert message: %v", err)
		}
	}
	fmt.Println("Created messages")

	// Create some reviews for completed work
	_, err = db.Exec(ctx, `
		INSERT INTO reviews (task_id, reviewer_id, reviewed_user_id, rating, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, tasks[7].ID, users[3].ID, users[2].ID, 5, 
	   "Charlie did an excellent job optimizing our database. Queries are now 10x faster!")
	
	if err != nil {
		log.Printf("Warning: Failed to insert review: %v", err)
	}

	fmt.Println("\n=== Test User Login Details ===")
	fmt.Println("All passwords are the username without the role + '123'")
	fmt.Println()
	for _, user := range users {
		fmt.Printf("Username: %s\n", user.Username)
		fmt.Printf("Email: %s\n", user.Email)
		fmt.Printf("Password: %s\n", user.Password)
		fmt.Printf("Full Name: %s\n", user.FullName)
		fmt.Println("---")
	}

	fmt.Println("\nTest data created successfully!")
}