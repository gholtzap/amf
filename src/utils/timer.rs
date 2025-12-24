use std::time::Duration;
use tokio::time::sleep;

pub struct Timer {
    duration: Duration,
}

impl Timer {
    pub fn new(duration: Duration) -> Self {
        Self { duration }
    }

    pub async fn wait(&self) {
        sleep(self.duration).await;
    }

    pub fn duration(&self) -> Duration {
        self.duration
    }
}

pub fn t3502(value: u32) -> Timer {
    Timer::new(Duration::from_secs(value as u64))
}

pub fn t3510(value: u32) -> Timer {
    Timer::new(Duration::from_secs(value as u64))
}

pub fn t3512(value: u32) -> Timer {
    Timer::new(Duration::from_secs(value as u64))
}

pub fn t3560(value: u32) -> Timer {
    Timer::new(Duration::from_secs(value as u64))
}

pub fn t3565(value: u32) -> Timer {
    Timer::new(Duration::from_secs(value as u64))
}
