import asyncio
import random
import time
import uuid
from enum import Enum
from typing import List, Dict, Any, Optional

import numpy as np
from fastapi import FastAPI, BackgroundTasks
from pydantic import BaseModel, Field
from typing_extensions import Annotated

app = FastAPI(title="Distributed Task Simulation Service")

# Task Type Enumeration
class TaskType(str, Enum):
    CPU_INTENSIVE = "cpu_intensive"
    IO_BOUND = "io_bound"
    MEMORY_INTENSIVE = "memory_intensive"
    NETWORK_INTENSIVE = "network_intensive"
    REAL_TIME = "real_time"
    DEPENDENT = "dependent"
    MIXED = "mixed"

class TaskPriority(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"

class Task(BaseModel):
    id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    type: TaskType
    priority: TaskPriority = TaskPriority.MEDIUM
    dependencies: List[str] = []
    estimated_duration: float = Field(ge=0.1, le=10.0)
    resource_requirements: Dict[str, float] = {
        "cpu": 0.0,
        "memory": 0.0,
        "network": 0.0
    }

class TaskResult(BaseModel):
    task_id: str
    status: str
    execution_time: float
    result: Optional[Any] = None

class TaskSimulator:
    @staticmethod
    def cpu_intensive_task(duration: float) -> int:
        """Simulate CPU-intensive task using matrix multiplication"""
        size = int(duration * 100)  # Scale matrix size with duration
        matrix1 = np.random.rand(size, size)
        matrix2 = np.random.rand(size, size)
        result = np.matmul(matrix1, matrix2)
        return int(np.sum(result))

    @staticmethod
    async def io_bound_task(duration: float) -> int:
        """Simulate I/O bound task with async sleep and file-like operations"""
        await asyncio.sleep(duration)
        return random.randint(1000, 9999)

    @staticmethod
    def memory_intensive_task(duration: float) -> int:
        """Simulate memory-intensive task by creating large data structures"""
        size = int(duration * 1_000_000)
        large_list = [random.random() for _ in range(size)]
        return len(large_list)

    @staticmethod
    async def network_intensive_task(duration: float) -> int:
        """Simulate network-intensive task with multiple async sleeps"""
        chunks = int(duration * 5)
        for _ in range(chunks):
            await asyncio.sleep(0.1)
        return random.randint(10000, 99999)

    @staticmethod
    async def real_time_task(deadline: float) -> bool:
        """Simulate real-time task with deadline constraint"""
        start_time = time.time()
        await asyncio.sleep(random.uniform(0.1, deadline))
        return time.time() - start_time <= deadline

    @staticmethod
    async def execute_task(task: Task) -> TaskResult:
        """Execute task based on its type"""
        start_time = time.time()
        """  """
        try:
            if task.type == TaskType.CPU_INTENSIVE:
                result = TaskSimulator.cpu_intensive_task(task.estimated_duration)
            elif task.type == TaskType.IO_BOUND:
                result = await TaskSimulator.io_bound_task(task.estimated_duration)
            elif task.type == TaskType.MEMORY_INTENSIVE:
                result = TaskSimulator.memory_intensive_task(task.estimated_duration)
            elif task.type == TaskType.NETWORK_INTENSIVE:
                result = await TaskSimulator.network_intensive_task(task.estimated_duration)
            elif task.type == TaskType.REAL_TIME:
                result = await TaskSimulator.real_time_task(task.estimated_duration)
            else:
                result = random.randint(1, 1000)
            
            execution_time = time.time() - start_time
            
            return TaskResult(
                task_id=task.id,
                status="completed",
                execution_time=execution_time,
                result=result
            )
        
        except Exception as e:
            return TaskResult(
                task_id=task.id,
                status="failed",
                execution_time=time.time() - start_time,
                result=str(e)
            )

# Task Generation Utility
class TaskGenerator:
    @staticmethod
    def generate_task_batch(
        num_tasks: int = 10, 
        task_types: List[TaskType] = None
    ) -> List[Task]:
        """Generate a batch of tasks with varied characteristics"""
        if task_types is None:
            task_types = list(TaskType)
        
        tasks = []
        for _ in range(num_tasks):
            task_type = random.choice(task_types)
            task = Task(
                type=task_type,
                priority=random.choice(list(TaskPriority)),
                estimated_duration=random.uniform(0.5, 5.0),
                resource_requirements={
                    "cpu": random.uniform(0.1, 1.0),
                    "memory": random.uniform(0.1, 1.0),
                    "network": random.uniform(0.1, 1.0)
                }
            )
            tasks.append(task)
        
        return tasks

# API Endpoints
@app.post("/tasks/generate", response_model=List[Task])
async def generate_tasks(
    num_tasks: Annotated[int, Field(ge=1, le=100)] = 10,
    task_types: Optional[List[TaskType]] = None
):
    """Generate a batch of tasks for scheduling simulation"""
    return TaskGenerator.generate_task_batch(num_tasks, task_types)

@app.post("/tasks/execute", response_model=TaskResult)
async def execute_task(task: Task, background_tasks: BackgroundTasks):
    """Execute a single task and return its result"""
    result = await TaskSimulator.execute_task(task)
    return result

@app.post("/tasks/batch-execute", response_model=List[TaskResult])
async def execute_batch_tasks(tasks: List[Task]):
    """Execute a batch of tasks concurrently"""
    results = await asyncio.gather(
        *[TaskSimulator.execute_task(task) for task in tasks]
    )
    return results

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000, workers=4)
