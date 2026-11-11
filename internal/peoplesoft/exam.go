package peoplesoft

// Exam represents a single exam entry from the PeopleSoft exam timetable page.
type Exam struct {
	Date        string // e.g., "Monday Jan 15,2027"
	Time        string // e.g., "2.00 PM to 4.30 PM"
	CourseCode  string // e.g., "FAK 1001"
	CourseName  string // e.g., "Sample Course Title"
	Location    string // e.g., "FAK BLDG Hall A"
	SeatNumber  string // optional seat number
	ExamType    string // e.g., "Final Exam"
}
