-- Fixed university curriculum: every course a student of this programme takes,
-- grouped by year and semester. Courses are added to a session from this list.

CREATE TABLE IF NOT EXISTS course_catalog (
  id          CHAR(36) PRIMARY KEY,
  code        VARCHAR(32) NOT NULL,
  title       VARCHAR(200) NOT NULL,
  credit      DECIMAL(4,2) NOT NULL,
  year_no     TINYINT NOT NULL,
  semester_no TINYINT NOT NULL,
  category    ENUM('CORE','OPTIONAL_I','OPTIONAL_II') NOT NULL DEFAULT 'CORE',
  sort_order  INT NOT NULL DEFAULT 0,
  UNIQUE KEY catalog_code_uidx (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO course_catalog (id, code, title, credit, year_no, semester_no, category, sort_order) VALUES
('cat-cse1101', 'CSE 1101', 'Computer Fundamentals', 3.00, 1, 1, 'CORE', 0),
('cat-cse1102', 'CSE 1102', 'Computer Fundamentals Sessional', 1.50, 1, 1, 'CORE', 1),
('cat-cse1103', 'CSE 1103', 'Structured Programming Language', 3.00, 1, 1, 'CORE', 2),
('cat-cse1104', 'CSE 1104', 'Structured Programming Language Sessional', 1.50, 1, 1, 'CORE', 3),
('cat-math1101', 'MATH 1101', 'Differential Calculus and Co-ordinate Geometry', 3.00, 1, 1, 'CORE', 4),
('cat-phy1101', 'PHY 1101', 'Physics', 3.00, 1, 1, 'CORE', 5),
('cat-hum1101', 'HUM 1101', 'Communicative English', 3.00, 1, 1, 'CORE', 6),
('cat-hum1102', 'HUM 1102', 'Communicative English Sessional', 0.75, 1, 1, 'CORE', 7),
('cat-cse1150', 'CSE 1150', 'Viva Voce', 0.75, 1, 1, 'CORE', 8),
('cat-me1200', 'ME 1200', 'Engineering Drawing', 1.00, 1, 2, 'CORE', 9),
('cat-cse1200', 'CSE 1200', 'Analytical Programming Sessional', 0.75, 1, 2, 'CORE', 10),
('cat-cse1201', 'CSE 1201', 'Object Oriented Programming', 3.00, 1, 2, 'CORE', 11),
('cat-cse1202', 'CSE 1202', 'Object Oriented Programming Sessional', 1.50, 1, 2, 'CORE', 12),
('cat-cse1203', 'CSE 1203', 'Discrete Mathematics', 3.00, 1, 2, 'CORE', 13),
('cat-eee1201', 'EEE 1201', 'Basic Electrical Engineering', 3.00, 1, 2, 'CORE', 14),
('cat-eee1202', 'EEE 1202', 'Basic Electrical Engineering Sessional', 1.50, 1, 2, 'CORE', 15),
('cat-math1201', 'MATH 1201', 'Integral Calculus, Differential Equation and Series Solution', 3.00, 1, 2, 'CORE', 16),
('cat-hum1201', 'HUM 1201', 'Economics', 2.00, 1, 2, 'CORE', 17),
('cat-cse1250', 'CSE 1250', 'Viva Voce', 0.75, 1, 2, 'CORE', 18),
('cat-cse2100', 'CSE 2100', 'Mobile Application Development Project', 1.50, 2, 1, 'CORE', 19),
('cat-cse2101', 'CSE 2101', 'Data Structures', 3.00, 2, 1, 'CORE', 20),
('cat-cse2102', 'CSE 2102', 'Data Structures Sessional', 1.50, 2, 1, 'CORE', 21),
('cat-cse2103', 'CSE 2103', 'Design Pattern and Java Programming', 3.00, 2, 1, 'CORE', 22),
('cat-cse2104', 'CSE 2104', 'Design Pattern and Java Programming Sessional', 0.75, 2, 1, 'CORE', 23),
('cat-eee2101', 'EEE 2101', 'Electronic Devices and Circuits', 3.00, 2, 1, 'CORE', 24),
('cat-eee2102', 'EEE 2102', 'Electronic Devices and Circuits Sessional', 0.75, 2, 1, 'CORE', 25),
('cat-math2101', 'MATH 2101', 'Vector, Matrices and Linear Algebra', 3.00, 2, 1, 'CORE', 26),
('cat-stat2101', 'STAT 2101', 'Elementary Statistics and Probability', 3.00, 2, 1, 'CORE', 27),
('cat-cse2150', 'CSE 2150', 'Viva Voce', 0.75, 2, 1, 'CORE', 28),
('cat-cse2200', 'CSE 2200', 'Hardware Project', 1.50, 2, 2, 'CORE', 29),
('cat-cse2201', 'CSE 2201', 'Algorithms', 3.00, 2, 2, 'CORE', 30),
('cat-cse2202', 'CSE 2202', 'Algorithms Sessional', 1.50, 2, 2, 'CORE', 31),
('cat-cse2203', 'CSE 2203', 'Theory of Computation', 2.00, 2, 2, 'CORE', 32),
('cat-cse2205', 'CSE 2205', 'Digital Systems', 3.00, 2, 2, 'CORE', 33),
('cat-cse2206', 'CSE 2206', 'Digital Systems Sessional', 1.50, 2, 2, 'CORE', 34),
('cat-math2201', 'MATH 2201', 'Complex Analysis, Laplace and Fourier Transforms', 3.00, 2, 2, 'CORE', 35),
('cat-stat2201', 'STAT 2201', 'Theory of Statistics', 3.00, 2, 2, 'CORE', 36),
('cat-cse2250', 'CSE 2250', 'Viva Voce', 0.75, 2, 2, 'CORE', 37),
('cat-cse3100', 'CSE 3100', 'Software Project I', 1.50, 3, 1, 'CORE', 38),
('cat-cse3101', 'CSE 3101', 'Computer Architecture and Organization', 3.00, 3, 1, 'CORE', 39),
('cat-cse3103', 'CSE 3103', 'Compiler Design', 3.00, 3, 1, 'CORE', 40),
('cat-cse3104', 'CSE 3104', 'Compiler Design Sessional', 0.75, 3, 1, 'CORE', 41),
('cat-cse3105', 'CSE 3105', 'Numerical Methods', 3.00, 3, 1, 'CORE', 42),
('cat-cse3106', 'CSE 3106', 'Numerical Methods Sessional', 0.75, 3, 1, 'CORE', 43),
('cat-cse3107', 'CSE 3107', 'Database Management Systems', 3.00, 3, 1, 'CORE', 44),
('cat-cse3108', 'CSE 3108', 'Database Management Systems Sessional', 1.50, 3, 1, 'CORE', 45),
('cat-ece3101', 'ECE 3101', 'Data Communication', 3.00, 3, 1, 'CORE', 46),
('cat-ece3102', 'ECE 3102', 'Data Communication Sessional', 0.75, 3, 1, 'CORE', 47),
('cat-cse3150', 'CSE 3150', 'Viva Voce', 0.75, 3, 1, 'CORE', 48),
('cat-cse3200', 'CSE 3200', 'Software Project II', 1.50, 3, 2, 'CORE', 49),
('cat-cse3201', 'CSE 3201', 'System Analysis and Design', 3.00, 3, 2, 'CORE', 50),
('cat-cse3203', 'CSE 3203', 'Operating Systems', 3.00, 3, 2, 'CORE', 51),
('cat-cse3204', 'CSE 3204', 'Operating Systems Sessional', 1.50, 3, 2, 'CORE', 52),
('cat-cse3205', 'CSE 3205', 'Web Engineering', 3.00, 3, 2, 'CORE', 53),
('cat-cse3206', 'CSE 3206', 'Web Engineering Sessional', 1.00, 3, 2, 'CORE', 54),
('cat-cse3207', 'CSE 3207', 'Digital Signal Processing', 3.00, 3, 2, 'CORE', 55),
('cat-cse3208', 'CSE 3208', 'Digital Signal Processing Sessional', 0.75, 3, 2, 'CORE', 56),
('cat-cse3209', 'CSE 3209', 'Microprocessors and Assembly Language', 3.00, 3, 2, 'CORE', 57),
('cat-cse3210', 'CSE 3210', 'Microprocessors and Assembly Language Sessional', 1.50, 3, 2, 'CORE', 58),
('cat-cse3250', 'CSE 3250', 'Viva Voce', 0.75, 3, 2, 'CORE', 59),
('cat-cse4100', 'CSE 4100', 'Thesis', 1.50, 4, 1, 'CORE', 60),
('cat-cse4101', 'CSE 4101', 'Software Engineering', 3.00, 4, 1, 'CORE', 61),
('cat-cse4102', 'CSE 4102', 'Software Project III', 1.50, 4, 1, 'CORE', 62),
('cat-cse4103', 'CSE 4103', 'Artificial Intelligence', 3.00, 4, 1, 'CORE', 63),
('cat-cse4104', 'CSE 4104', 'Artificial Intelligence Sessional', 0.75, 4, 1, 'CORE', 64),
('cat-cse4105', 'CSE 4105', 'Digital Image Processing', 3.00, 4, 1, 'CORE', 65),
('cat-cse4106', 'CSE 4106', 'Digital Image Processing Sessional', 1.50, 4, 1, 'CORE', 66),
('cat-optional-i', 'OPTIONAL-I', 'Optional Course I', 3.00, 4, 1, 'OPTIONAL_I', 67),
('cat-hum4101', 'HUM 4101', 'Sociology and Bangladesh Studies', 3.00, 4, 1, 'CORE', 68),
('cat-cse4150', 'CSE 4150', 'Viva Voce', 0.75, 4, 1, 'CORE', 69),
('cat-cse4107', 'CSE 4107', 'Computer Simulation and Modeling', 3.00, 4, 1, 'OPTIONAL_I', 70),
('cat-cse4109', 'CSE 4109', 'Multimedia Technology', 3.00, 4, 1, 'OPTIONAL_I', 71),
('cat-cse4111', 'CSE 4111', 'Basic Graph Theory', 3.00, 4, 1, 'OPTIONAL_I', 72),
('cat-cse4113', 'CSE 4113', 'Parallel and Distributed Processing', 3.00, 4, 1, 'OPTIONAL_I', 73),
('cat-cse4115', 'CSE 4115', 'Data Mining', 3.00, 4, 1, 'OPTIONAL_I', 74),
('cat-cse4117', 'CSE 4117', 'Computer Vision', 3.00, 4, 1, 'OPTIONAL_I', 75),
('cat-cse4119', 'CSE 4119', 'Machine Learning', 3.00, 4, 1, 'OPTIONAL_I', 76),
('cat-cse4200', 'CSE 4200', 'Thesis', 3.00, 4, 2, 'CORE', 77),
('cat-cse4201', 'CSE 4201', 'Computer Networks', 3.00, 4, 2, 'CORE', 78),
('cat-cse4202', 'CSE 4202', 'Computer Networks Sessional', 1.50, 4, 2, 'CORE', 79),
('cat-cse4203', 'CSE 4203', 'Computer Graphics', 3.00, 4, 2, 'CORE', 80),
('cat-cse4204', 'CSE 4204', 'Computer Graphics Sessional', 1.50, 4, 2, 'CORE', 81),
('cat-cse4205', 'CSE 4205', 'Interfacing and Microcontrollers', 3.00, 4, 2, 'CORE', 82),
('cat-cse4206', 'CSE 4206', 'Interfacing and Microcontrollers Sessional', 0.75, 4, 2, 'CORE', 83),
('cat-optional-ii', 'OPTIONAL-II', 'Optional Course II', 3.00, 4, 2, 'OPTIONAL_II', 84),
('cat-hum4201', 'HUM 4201', 'Industrial Management and Accounting', 3.00, 4, 2, 'CORE', 85),
('cat-cse4250', 'CSE 4250', 'Viva Voce', 0.75, 4, 2, 'CORE', 86),
('cat-cse4207', 'CSE 4207', 'Cryptography and Network Security', 3.00, 4, 2, 'OPTIONAL_II', 87),
('cat-cse4209', 'CSE 4209', 'VLSI Design', 3.00, 4, 2, 'OPTIONAL_II', 88),
('cat-cse4211', 'CSE 4211', 'Wireless Communication', 3.00, 4, 2, 'OPTIONAL_II', 89),
('cat-cse4213', 'CSE 4213', 'Computational Geometry', 3.00, 4, 2, 'OPTIONAL_II', 90),
('cat-cse4215', 'CSE 4215', 'Bioinformatics', 3.00, 4, 2, 'OPTIONAL_II', 91),
('cat-cse4217', 'CSE 4217', 'Human Computer Interaction', 3.00, 4, 2, 'OPTIONAL_II', 92),
('cat-cse4219', 'CSE 4219', 'Knowledge Engineering', 3.00, 4, 2, 'OPTIONAL_II', 93);


-- A course copied from the catalogue carries its credit; marks stay out of 100.
ALTER TABLE courses ADD COLUMN credit DECIMAL(4,2) NOT NULL DEFAULT 0 AFTER title;
ALTER TABLE courses ADD COLUMN catalog_id CHAR(36) NULL AFTER credit;

-- Deleting a session removes its courses and students, and their scripts too.
ALTER TABLE students DROP FOREIGN KEY students_session_fk;
ALTER TABLE students ADD CONSTRAINT students_session_fk FOREIGN KEY (session_id) REFERENCES academic_sessions(id) ON DELETE CASCADE;

ALTER TABLE courses DROP FOREIGN KEY courses_session_fk;
ALTER TABLE courses ADD CONSTRAINT courses_session_fk FOREIGN KEY (session_id) REFERENCES academic_sessions(id) ON DELETE CASCADE;

ALTER TABLE scripts DROP FOREIGN KEY scripts_course_fk;
ALTER TABLE scripts ADD CONSTRAINT scripts_course_fk FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE;

ALTER TABLE scripts DROP FOREIGN KEY scripts_student_fk;
ALTER TABLE scripts ADD CONSTRAINT scripts_student_fk FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;
