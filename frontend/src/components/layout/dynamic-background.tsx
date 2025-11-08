import React from 'react';

// TODO: Заменить на реальный fetch к Unsplash API или S3
const BACKGROUND_IMAGE_URL = 'https://picsum.photos/3840/2160';

export const DynamicBackground: React.FC = () => {
  return (
    <>
      <div
        className="filter-[blur(4px)_brightness(0.8)] dark:filter-[blur(4px)_brightness(0.4)] fixed inset-0 z-0 bg-cover bg-center bg-no-repeat"
        style={{
          backgroundImage: `url(${BACKGROUND_IMAGE_URL})`,
          transform: 'scale(1.1)',
        }}
        aria-hidden="true"
      />
    </>
  );
};
