"""Technical illustration for the report, not application image processing."""
from pathlib import Path
import os
os.environ.setdefault('MPLCONFIGDIR','/tmp/lb4-matplotlib')
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import numpy as np
root=Path(__file__).resolve().parents[1]
w=280;x=320;y=240
vs=np.array([[x,y-w/np.sqrt(3)],[x-w/2,y+w/(2*np.sqrt(3))],[x+w/2,y+w/(2*np.sqrt(3))]])
fig,ax=plt.subplots(figsize=(7,6),layout='constrained')
for v,start,col in zip(vs,[60,300,180],['#2660a6','#20946c','#a54880']):
 t=np.deg2rad(np.linspace(start,start+60,1001));ax.plot(v[0]+w*np.cos(t),v[1]+w*np.sin(t),color=col,lw=2.2)
ax.plot(*np.vstack([vs,vs[0]]).T,'--',color='#999',lw=1)
ax.scatter(*vs.T,color='#222',s=25)
for p,label,off in zip(vs,['A','B','C'],[(8,-7),(-17,12),(10,12)]):ax.annotate(label,p,xytext=off,textcoords='offset points',fontsize=13)
ax.scatter([x],[y],color='#222',s=18);ax.text(x+8,y,'O (320; 240)',fontsize=11)
ax.annotate('',xy=(180,392),xytext=(460,392),arrowprops={'arrowstyle':'<->','color':'#444'})
ax.text(320,414,'w = 280',ha='center',fontsize=12)
ax.set_xlim(140,500);ax.set_ylim(435,45);ax.set_aspect('equal');ax.set_xlabel('X, пиксели');ax.set_ylabel('Y, пиксели');ax.grid(alpha=.2)
fig.savefig(root/'report/geometry.png',dpi=180);fig.savefig(root/'report/geometry.svg');plt.close(fig)
