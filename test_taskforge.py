import requests
import json
import time
import sys
from datetime import datetime, timedelta
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE_URL = "http://192.168.0.103:8188"
API_URL = f"{BASE_URL}/v1"

class TaskForgeMassTester:
    def __init__(self):
        self.job_ids = []
        self.success_count = 0
        self.fail_count = 0
    
    def print_header(self, text):
        print("\n" + "="*60)
        print(text)
        print("="*60 + "\n")
    
    def print_info(self, text):
        print(f"[INFO] {text}")
    
    def print_progress(self, current, total, text=""):
        percent = (current / total) * 100
        bar_length = 40
        filled = int(bar_length * current // total)
        bar = '=' * filled + '-' * (bar_length - filled)
        print(f"\r[{bar}] {percent:.1f}% ({current}/{total}) {text}", end="")
        sys.stdout.flush()
    
    def get_stats(self):
        try:
            response = requests.get(f"{API_URL}/stats", timeout=5)
            if response.status_code == 200:
                return response.json()
            return None
        except:
            return None
    
    def create_job(self, job_type="test", payload=None, priority="default", delay_seconds=None):
        if payload is None:
            payload = {"timestamp": datetime.now().isoformat()}
        
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
                timeout=10
            )
            if response.status_code == 201:
                job_id = response.json().get("id")
                return job_id
            else:
                return None
        except:
            return None
    
    def create_jobs_batch(self, count, priority="default", job_type="mass_test", delay_seconds=None):
        """Create jobs in batch using threading"""
        self.print_info(f"Creating {count} {priority} priority jobs...")
        
        def create_single(job_index):
            payload = {"index": job_index, "priority": priority}
            job_id = self.create_job(job_type, payload, priority, delay_seconds)
            return job_id
        
        created = 0
        failed = 0
        
        # Use ThreadPoolExecutor for parallel creation
        with ThreadPoolExecutor(max_workers=50) as executor:
            futures = []
            for i in range(count):
                future = executor.submit(create_single, i)
                futures.append(future)
            
            for i, future in enumerate(as_completed(futures)):
                result = future.result()
                if result:
                    self.job_ids.append(result)
                    created += 1
                else:
                    failed += 1
                
                if (i + 1) % 100 == 0 or (i + 1) == count:
                    self.print_progress(i + 1, count, f"Created: {created}, Failed: {failed}")
        
        print()
        self.print_info(f"Created {created} jobs, Failed {failed} jobs")
        self.success_count += created
        self.fail_count += failed
        return created, failed
    
    def print_stats_table(self, stats):
        if not stats:
            print("[ERROR] No stats available")
            return
        
        print("\n" + "-"*40)
        print("QUEUE STATISTICS")
        print("-"*40)
        print(f"High Priority:    {stats.get('high', 0)}")
        print(f"Default Priority: {stats.get('default', 0)}")
        print(f"Low Priority:     {stats.get('low', 0)}")
        print(f"Delayed:          {stats.get('delayed', 0)}")
        print(f"Dead Letter Queue: {stats.get('dlq', 0)}")
        print("-"*40)
        total = sum(stats.values())
        print(f"TOTAL JOBS:       {total}")
        print("-"*40 + "\n")
    
    def monitor_stats(self, duration=30, interval=2):
        """Monitor stats for a duration"""
        self.print_header(f"MONITORING STATS (every {interval}s for {duration}s)")
        
        for i in range(0, duration, interval):
            stats = self.get_stats()
            if stats:
                total = sum(stats.values())
                print(f"[{i}s] High:{stats['high']:>4} Default:{stats['default']:>4} "
                      f"Low:{stats['low']:>4} Delayed:{stats['delayed']:>4} "
                      f"DLQ:{stats['dlq']:>4} Total:{total:>4}")
            else:
                print(f"[{i}s] No stats available")
            time.sleep(interval)
    
    def test_priority_distribution(self):
        """Test 10,000+ jobs across all priorities"""
        self.print_header("MASS TEST: 10,000+ Jobs")
        
        total_jobs = 10000
        
        # 3000 High priority
        self.create_jobs_batch(3000, "high")
        
        # 4000 Default priority
        self.create_jobs_batch(4000, "default")
        
        # 3000 Low priority
        self.create_jobs_batch(3000, "low")
        
        stats = self.get_stats()
        self.print_stats_table(stats)
        
        return stats
    
    def test_delayed_batch(self):
        """Test delayed jobs"""
        self.print_header("DELAYED JOBS TEST")
        
        count = 1000
        self.print_info(f"Creating {count} delayed jobs (30 second delay)...")
        
        created, failed = self.create_jobs_batch(
            count, 
            "default", 
            "delayed_test",
            delay_seconds=30
        )
        
        stats = self.get_stats()
        self.print_stats_table(stats)
        
        return stats
    
    def get_system_stats(self):
        """Get system stats including workers"""
        try:
            response = requests.get(f"{BASE_URL}/dashboard/stats", timeout=5)
            if response.status_code == 200:
                data = response.json()
                print("\nSYSTEM STATUS")
                print("-"*40)
                
                workers = data.get('workers', [])
                print(f"Active Workers: {len(workers)}")
                for worker in workers:
                    print(f"  - {worker.get('id', 'unknown')[:20]}... "
                          f"Status: {worker.get('status', 'unknown')} "
                          f"Jobs: {worker.get('active_jobs', 0)}")
                
                recent = data.get('recent_jobs', {})
                print("\nRecent Jobs:")
                for status, jobs in recent.items():
                    if jobs:
                        print(f"  - {status}: {len(jobs)} jobs")
                
                return data
            return None
        except:
            return None
    
    def run_mass_test(self):
        """Run complete mass test"""
        print("\n" + "="*60)
        print("TASKFORGE MASS TEST - 10,000+ JOBS")
        print("="*60 + "\n")
        
        self.print_info("Checking connection...")
        stats = self.get_stats()
        if not stats:
            self.print_info("Cannot connect to TaskForge. Make sure it's running.")
            sys.exit(1)
        
        self.print_info(f"Connected to {BASE_URL}")
        self.print_stats_table(stats)
        
        self.test_priority_distribution()
        
        self.print_info("Waiting for workers to start processing...")
        time.sleep(5)
        
        self.monitor_stats(20, 2)
        
        self.test_delayed_batch()
        
        self.get_system_stats()
        
        self.print_header("TEST SUMMARY")
        print(f"Total jobs created: {len(self.job_ids)}")
        print(f"Successful creations: {self.success_count}")
        print(f"Failed creations: {self.fail_count}")
        if self.success_count + self.fail_count > 0:
            print(f"Success rate: {(self.success_count / (self.success_count + self.fail_count) * 100):.2f}%")
        
        stats = self.get_stats()
        if stats:
            print(f"\nCurrent queue totals:")
            print(f"  High: {stats.get('high', 0)}")
            print(f"  Default: {stats.get('default', 0)}")
            print(f"  Low: {stats.get('low', 0)}")
            print(f"  Delayed: {stats.get('delayed', 0)}")
            print(f"  DLQ: {stats.get('dlq', 0)}")
            print(f"  Total: {sum(stats.values())}")
        
        print("\n" + "="*60)
        print("MASS TEST COMPLETE")
        print("="*60 + "\n")

if __name__ == "__main__":
    tester = TaskForgeMassTester()
    tester.run_mass_test()