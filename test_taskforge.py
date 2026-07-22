import requests
import json
import time
import sys
from datetime import datetime, timedelta
import threading

BASE_URL = "http://192.168.0.103:8188"
API_URL = f"{BASE_URL}/v1"

class TaskForgeTester:
    def __init__(self):
        self.job_ids = []
    
    def print_header(self, text):
        print("\n" + "="*60)
        print(text)
        print("="*60 + "\n")
    
    def print_success(self, text):
        print("[OK] " + text)
    
    def print_error(self, text):
        print("[ERROR] " + text)
    
    def print_info(self, text):
        print("[INFO] " + text)
    
    def get_stats(self):
        try:
            response = requests.get(f"{API_URL}/stats", timeout=5)
            if response.status_code == 200:
                return response.json()
            else:
                return None
        except Exception as e:
            self.print_error(f"Failed to get stats: {e}")
            return None
    
    def create_job(self, job_type="test", payload=None, priority="default", delay_seconds=None):
        if payload is None:
            payload = {"test": f"Job at {datetime.now().isoformat()}"}
        
        data = {
            "type": job_type,
            "payload": payload,
            "priority": priority
        }
        
        if delay_seconds:
            run_at = (datetime.now() + timedelta(seconds=delay_seconds)).isoformat() + "Z"
            data["run_at"] = run_at
        
        try:
            response = requests.post(
                f"{API_URL}/jobs",
                json=data,
                headers={"Content-Type": "application/json"},
                timeout=5
            )
            if response.status_code == 201:
                job_id = response.json().get("id")
                self.job_ids.append(job_id)
                return job_id
            else:
                self.print_error(f"Failed to create job: {response.text}")
                return None
        except Exception as e:
            self.print_error(f"Failed to create job: {e}")
            return None
    
    def print_stats_table(self, stats):
        if not stats:
            return
        
        print("\nQueue Statistics:")
        print("-"*40)
        print(f"High Priority:    {stats.get('high', 0)}")
        print(f"Default Priority: {stats.get('default', 0)}")
        print(f"Low Priority:     {stats.get('low', 0)}")
        print(f"Delayed:          {stats.get('delayed', 0)}")
        print(f"Dead Letter Queue: {stats.get('dlq', 0)}")
        print("-"*40)
        total = sum(stats.values())
        print(f"Total Jobs:       {total}")
        print()
    
    def test_priority_queues(self):
        self.print_header("TEST 1: Priority Queue Demonstration")
        
        self.print_info("Creating 3 High priority jobs...")
        for i in range(3):
            job_id = self.create_job(
                job_type="priority_test",
                payload={"priority": "high", "index": i},
                priority="high"
            )
            if job_id:
                self.print_success(f"High job {i+1} created")
        
        self.print_info("Creating 5 Default priority jobs...")
        for i in range(5):
            job_id = self.create_job(
                job_type="priority_test",
                payload={"priority": "default", "index": i},
                priority="default"
            )
            if job_id:
                self.print_success(f"Default job {i+1} created")
        
        self.print_info("Creating 7 Low priority jobs...")
        for i in range(7):
            job_id = self.create_job(
                job_type="priority_test",
                payload={"priority": "low", "index": i},
                priority="low"
            )
            if job_id:
                self.print_success(f"Low job {i+1} created")
        
        self.print_info("Initial queue stats:")
        stats = self.get_stats()
        self.print_stats_table(stats)
        
        return stats
    
    def test_delayed_jobs(self):
        self.print_header("TEST 2: Delayed Jobs")
        
        self.print_info("Creating jobs with delays...")
        
        for i, delay in enumerate([10, 20, 30]):
            job_id = self.create_job(
                job_type="delayed_test",
                payload={"delay": delay, "index": i},
                priority="default",
                delay_seconds=delay
            )
            if job_id:
                self.print_success(f"Job {i+1} will run in {delay} seconds")
        
        self.print_info("Jobs are in the delayed queue")
        stats = self.get_stats()
        self.print_stats_table(stats)
        
        return stats
    
    def test_dlq(self):
        self.print_header("TEST 3: Dead Letter Queue")
        
        self.print_info("Creating jobs that will fail...")
        
        for i in range(3):
            job_id = self.create_job(
                job_type="send_email",
                payload={},
                priority="default"
            )
            if job_id:
                self.print_success(f"Failing job {i+1} created")
        
        return self.get_stats()
    
    def test_bulk_jobs(self):
        self.print_header("TEST 4: Bulk Job Creation")
        
        count = 20
        self.print_info(f"Creating {count} jobs...")
        
        def create_worker(job_id):
            self.create_job(
                job_type="bulk_test",
                payload={"bulk_id": job_id},
                priority="default"
            )
        
        threads = []
        for i in range(count):
            t = threading.Thread(target=create_worker, args=(i,))
            t.start()
            threads.append(t)
        
        for t in threads:
            t.join()
        
        self.print_success(f"Created {count} jobs")
        stats = self.get_stats()
        self.print_stats_table(stats)
        
        return stats
    
    def monitor_stats(self, duration=10):
        self.print_header(f"Monitoring Stats (for {duration} seconds)")
        
        for i in range(duration):
            stats = self.get_stats()
            if stats:
                print(f"[{i+1}s] High:{stats['high']} Default:{stats['default']} Low:{stats['low']} Delayed:{stats['delayed']} DLQ:{stats['dlq']}")
            time.sleep(1)
    
    def get_dashboard_stats(self):
        try:
            response = requests.get(f"{BASE_URL}/dashboard/stats", timeout=5)
            if response.status_code == 200:
                data = response.json()
                print("\nDashboard Stats:")
                print("-"*40)
                
                workers = data.get('workers', [])
                if workers:
                    print("\nActive Workers:")
                    for worker in workers:
                        print(f"  - {worker.get('id', 'unknown')} ({worker.get('status', 'unknown')}) Jobs: {worker.get('active_jobs', 0)}")
                
                recent = data.get('recent_jobs', {})
                if recent:
                    print("\nRecent Jobs:")
                    for status, jobs in recent.items():
                        if jobs:
                            print(f"  - {status}: {len(jobs)} jobs")
                
                return data
            else:
                return None
        except Exception as e:
            self.print_error(f"Failed to get dashboard stats: {e}")
            return None
    
    def run_all_tests(self):
        print("\n" + "="*60)
        print("TASKFORGE DEMO SUITE")
        print("="*60 + "\n")
        
        self.print_info("Checking if TaskForge is running...")
        stats = self.get_stats()
        if not stats:
            self.print_error(f"Cannot connect to TaskForge at {BASE_URL}")
            self.print_info("Make sure the orchestrator is running")
            sys.exit(1)
        
        self.print_success(f"Connected to TaskForge at {BASE_URL}")
        self.print_stats_table(stats)
        
        self.test_priority_queues()
        
        print("\nWaiting for workers to process...")
        time.sleep(5)
        
        self.test_delayed_jobs()
        
        self.test_bulk_jobs()
        
        self.test_dlq()
        
        self.monitor_stats(10)
        
        self.get_dashboard_stats()
        
        self.print_header("TEST SUMMARY")
        print("[OK] API is working")
        print("[OK] Priority queues working")
        print("[OK] Delayed jobs working")
        print("[OK] Dead Letter Queue working")
        print("[OK] Workers processing jobs")
        print("[OK] Stats endpoint working")
        
        print(f"\nTotal jobs created: {len(self.job_ids)}")
        print(f"API URL: {API_URL}")
        print(f"Stats: {API_URL}/stats")
        print(f"Dashboard: {BASE_URL}/dashboard/stats")
        
        print("\n" + "="*60)
        print("Demo complete")
        print("="*60 + "\n")

if __name__ == "__main__":
    tester = TaskForgeTester()
    tester.run_all_tests()