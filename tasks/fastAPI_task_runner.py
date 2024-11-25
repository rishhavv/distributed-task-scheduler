from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import time
import os
import math
import random

# Initialize FastAPI app
app = FastAPI()

# Task input model
class TaskInput(BaseModel):
    task_type: str  # "read", "write", "delay", "compute", "dependent", or "mixed"
    content: str = None  # Used for "write"
    delay: int = 0  # Used for "delay" in seconds
    compute_size: int = 0  # Number of iterations for "compute"
    dependency: str = None  # Simulates dependent tasks

# Base directory for simulated files
BASE_DIR = "task_files"
os.makedirs(BASE_DIR, exist_ok=True)

@app.post("/perform_task")
async def perform_task(task: TaskInput):
    """
    Endpoint to simulate various tasks:
    - task_type = "read": Read a file.
    - task_type = "write": Write content to a file.
    - task_type = "delay": Simulate a delay.
    - task_type = "compute": Perform a CPU-intensive computation.
    - task_type = "dependent": Simulate task dependency resolution.
    - task_type = "mixed": Combination of delay, I/O, and compute tasks.
    """
    try:
        if task.task_type == "read":
            file_path = os.path.join(BASE_DIR, "sample.txt")
            if not os.path.exists(file_path):
                return {"status": "failed", "reason": "File not found for reading"}
            with open(file_path, "r") as f:
                content = f.read()
            return {"status": "success", "content": content}
        
        elif task.task_type == "write":
            file_path = os.path.join(BASE_DIR, "sample.txt")
            with open(file_path, "w") as f:
                f.write(task.content or "Default content")
            return {"status": "success", "message": "File written successfully"}
        
        elif task.task_type == "delay":
            time.sleep(task.delay)
            return {"status": "success", "message": f"Delay of {task.delay} seconds completed"}
        
        elif task.task_type == "compute":
            result = sum(math.sqrt(i) for i in range(task.compute_size))
            return {"status": "success", "message": f"Computed sum of sqrt(0) to sqrt({task.compute_size - 1})", "result": result}
        
        elif task.task_type == "dependent":
            if not task.dependency:
                return {"status": "failed", "reason": "No dependency provided"}
            # Simulate checking for dependency resolution
            dependency_file = os.path.join(BASE_DIR, f"{task.dependency}.txt")
            if not os.path.exists(dependency_file):
                return {"status": "failed", "reason": f"Dependency {task.dependency} not resolved"}
            return {"status": "success", "message": f"Dependency {task.dependency} resolved"}
        
        elif task.task_type == "mixed":
            # Simulate a combination of tasks
            time.sleep(task.delay or 1)  # Default delay of 1 second if not provided
            file_path = os.path.join(BASE_DIR, "mixed_task.txt")
            with open(file_path, "w") as f:
                f.write("Mixed task content")
            result = sum(random.random() for _ in range(1000))
            return {
                "status": "success",
                "message": "Mixed task completed",
                "details": {"delay": task.delay, "file_written": file_path, "compute_result": result}
            }
        
        else:
            return {"status": "failed", "reason": "Invalid task_type provided"}
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

# Run the app
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=4444, workers=4)
